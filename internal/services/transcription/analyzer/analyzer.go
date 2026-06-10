//go:build completion

// Package analyzer 提供音频转 MIDI 期间的辅助分析：
//   - 音频质量检测（PCM 削波/静音/动态范围）
//   - BPM 估计（基于 note onset interval 聚类）
//   - 调性 / 音阶分析（基于 pitch class histogram + Krumhansl-Schmuckler 相关性）
package analyzer

import (
	"sort"

	"YyslsPlayer/internal/services/transcription/engine"
)

// AudioQuality 音频质量分析结果。
type AudioQuality struct {
	Score           float64 `json:"score"`
	ClippingDetected bool   `json:"clippingDetected"`
	SilenceRatio    float64 `json:"silenceRatio"`
}

// AnalyzeAudioQuality 基于 PCM 样本做质量分析。
// pcmSamples 为 float32 归一化采样（-1..1），sampleRate 为采样率。
func AnalyzeAudioQuality(pcmSamples []float32, sampleRate int) AudioQuality {
	if len(pcmSamples) == 0 {
		return AudioQuality{Score: 0}
	}

	total := len(pcmSamples)
	silentCount := 0
	clipCount := 0
	var maxAbs float32

	for _, s := range pcmSamples {
		abs := s
		if abs < 0 {
			abs = -abs
		}
		if abs > maxAbs {
			maxAbs = abs
		}
		if abs < 0.001 {
			silentCount++
		}
		if abs > 0.999 {
			clipCount++
		}
	}

	silenceRatio := float64(silentCount) / float64(total)
	clippingDetected := float64(clipCount)/float64(total) > 0.001

	score := 100.0
	// 静音比例惩罚
	if silenceRatio > 0.5 {
		score -= 40
	} else if silenceRatio > 0.2 {
		score -= 15
	}
	// 削波惩罚
	if clippingDetected {
		score -= 25
	}
	// 动态范围偏低惩罚
	if maxAbs < 0.1 {
		score -= 20
	}

	if score < 0 {
		score = 0
	}

	return AudioQuality{
		Score:            score,
		ClippingDetected:  clippingDetected,
		SilenceRatio:     silenceRatio,
	}
}

// BPMEstimation BPM 估计结果。
type BPMEstimation struct {
	BPM        float64 `json:"bpm"`
	Confidence float64 `json:"confidence"`
}

// EstimateBPM 基于 note onset 时间聚类估计 BPM。
// 扫描相邻 onset 间隔，找最常见间隔对应的 BPM。
func EstimateBPM(notes []engine.RawNote) BPMEstimation {
	if len(notes) < 3 {
		return BPMEstimation{BPM: 120, Confidence: 0}
	}

	// 按 onset 时间排序
	onsets := make([]int64, len(notes))
	for i, n := range notes {
		onsets[i] = n.StartMs
	}
	sort.Slice(onsets, func(i, j int) bool { return onsets[i] < onsets[j] })

	// 统计相邻 onset 间隔频率
	intervalCounts := make(map[int64]int)
	for i := 1; i < len(onsets); i++ {
		diff := onsets[i] - onsets[i-1]
		if diff > 0 && diff < 5000 { // 忽略过长间隔（休止）
			// 量化到 10ms 精度
			quantized := (diff / 10) * 10
			if quantized < 50 {
				quantized = diff
			}
			intervalCounts[quantized]++
		}
	}

	if len(intervalCounts) == 0 {
		return BPMEstimation{BPM: 120, Confidence: 0}
	}

	// 找最常见间隔
	var bestInterval int64
	var bestCount int
	for interval, count := range intervalCounts {
		if count > bestCount {
			bestCount = count
			bestInterval = interval
		}
	}

	if bestInterval <= 0 {
		return BPMEstimation{BPM: 120, Confidence: 0}
	}

	// 间隔 → BPM
	bpm := 60000.0 / float64(bestInterval)
	// 约束到合理范围
	if bpm < 30 {
		bpm *= 2
	}
	if bpm > 300 {
		bpm /= 2
	}

	totalIntervals := len(onsets) - 1
	confidence := float64(bestCount) / float64(totalIntervals)
	if confidence > 1 {
		confidence = 1
	}

	return BPMEstimation{BPM: bpm, Confidence: confidence}
}

// ===== Key/Scale Analysis =====

// Krumhansl-Schmuckler 调性特征向量（major 和 minor）
// 值来自 Krumhansl & Kessler (1982) 认知实验归一化数据
var ksMajor = [12]float64{
	6.35, 2.23, 3.48, 2.33, 4.38, 4.09, // C, C#, D, D#, E, F
	2.52, 5.19, 2.39, 3.66, 2.29, 2.88, // F#, G, G#, A, A#, B
}

var ksMinor = [12]float64{
	6.33, 2.68, 3.52, 5.38, 2.60, 3.53, // C, C#, D, D#, E, F
	2.54, 4.75, 3.98, 2.69, 3.34, 3.17, // F#, G, G#, A, A#, B
}

// PitchClassHistogram 构建 12 音 pitch class 直方图。
func PitchClassHistogram(notes []engine.RawNote) []float64 {
	hist := make([]float64, 12)
	if len(notes) == 0 {
		return hist
	}
	for _, n := range notes {
		pc := n.Note % 12
		if pc < 0 {
			pc += 12
		}
		hist[pc] += n.Confidence
	}
	// 归一化
	total := 0.0
	for _, v := range hist {
		total += v
	}
	if total > 0 {
		for i := range hist {
			hist[i] /= total
		}
	}
	return hist
}

// KeyCandidate 调性候选。
type KeyCandidate struct {
	Tonic      string  `json:"tonic"`
	Mode       string  `json:"mode"`
	Confidence float64 `json:"confidence"`
}

// EstimateKey 基于 pitch class histogram 估计调性。
// 返回候选列表（按相关性降序）。
func EstimateKey(hist []float64) []KeyCandidate {
	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	type candidate struct {
		tonic      int
		mode       string
		correlation float64
	}

	var candidates []candidate
	for tonic := 0; tonic < 12; tonic++ {
		corrMajor := correlation(hist, ksMajor[:], tonic)
		corrMinor := correlation(hist, ksMinor[:], tonic)
		candidates = append(candidates,
			candidate{tonic, "major", corrMajor},
			candidate{tonic, "minor", corrMinor},
		)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].correlation > candidates[j].correlation
	})

	result := make([]KeyCandidate, 0, 4)
	for _, c := range candidates {
		if c.correlation > 0.5 && len(result) < 4 {
			result = append(result, KeyCandidate{
				Tonic:      noteNames[c.tonic],
				Mode:       c.mode,
				Confidence: c.correlation,
			})
		}
	}

	if len(result) == 0 {
		// 至少返回最佳候选
		best := candidates[0]
		result = append(result, KeyCandidate{
			Tonic:      noteNames[best.tonic],
			Mode:       best.mode,
			Confidence: best.correlation,
		})
	}

	return result
}

// correlation 计算 Pearson 相关系数（circular shift 版本）。
func correlation(a, b []float64, shift int) float64 {
	n := len(a)
	if n != 12 || len(b) != 12 {
		return 0
	}

	// 旋转 b 以匹配 tonic shift
	rotated := make([]float64, n)
	for i := 0; i < n; i++ {
		rotated[i] = b[(i-shift+n)%n]
	}

	var sumA, sumB, sumAB, sumA2, sumB2 float64
	for i := 0; i < n; i++ {
		sumA += a[i]
		sumB += rotated[i]
		sumAB += a[i] * rotated[i]
		sumA2 += a[i] * a[i]
		sumB2 += rotated[i] * rotated[i]
	}

	meanA := sumA / float64(n)
	meanB := sumB / float64(n)

	cov := sumAB/float64(n) - meanA*meanB
	varA := sumA2/float64(n) - meanA*meanA
	varB := sumB2/float64(n) - meanB*meanB

	if varA <= 0 || varB <= 0 {
		return 0
	}

	return cov / (sqrt(varA) * sqrt(varB))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
