//go:build completion

// Package midiout 提供 SMF MIDI 文件写出和导入现有 MIDI 链路的能力。
package midiout

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/services/transcription/engine"
	"YyslsPlayer/internal/storage"
)

// SMFWriter 将音符列表写出为标准 MIDI 文件。
type SMFWriter struct{}

// NewSMFWriter 创建 SMF MIDI 写出器。
func NewSMFWriter() *SMFWriter {
	return &SMFWriter{}
}

// Write 将音符列表写出为 SMF MIDI 文件。
//
// 使用手动二进制构造 SMF format 1, PPQ 480, tempo track + note track。
// 不依赖外部库，保持构建纯度。
func (w *SMFWriter) Write(notes []engine.RawNote, bpm float64, targetPath string) error {
	if err := os.MkdirAll(dirOf(targetPath), 0755); err != nil {
		return fmt.Errorf("midi.write_failed: %w", err)
	}

	ppq := uint16(480)
	if bpm <= 0 {
		bpm = 120
	}
	tempo := uint32(60000000.0 / bpm) // microseconds per quarter

	// 构建 SMF 格式 1 二进制
	// Track 0: tempo + time signature
	// Track 1: note events

	track0 := buildTrack0(tempo, ppq)
	track1 := buildTrack1(notes, ppq, tempo)

	data := assembleSMF(track0, track1, ppq)

	return os.WriteFile(targetPath, data, 0644)
}

// buildTrack0 构建 tempo track。
func buildTrack0(tempo uint32, ppq uint16) []byte {
	var buf writeBuffer

	// delta time 0
	buf.writeVarLen(0)

	// Tempo meta event: FF 51 03 tt tt tt
	buf.writeByte(0xFF)
	buf.writeByte(0x51)
	buf.writeByte(0x03)
	buf.writeByte(byte(tempo >> 16))
	buf.writeByte(byte(tempo >> 8))
	buf.writeByte(byte(tempo))

	// Time signature: FF 58 04 nn dd cc bb
	buf.writeVarLen(0)
	buf.writeByte(0xFF)
	buf.writeByte(0x58)
	buf.writeByte(0x04)
	buf.writeByte(4) // numerator
	buf.writeByte(2) // denominator (2=quarter)
	buf.writeByte(24)
	buf.writeByte(8)

	// End of track: FF 2F 00
	buf.writeVarLen(0)
	buf.writeByte(0xFF)
	buf.writeByte(0x2F)
	buf.writeByte(0x00)

	length := buf.len()
	result := make([]byte, 0, 8+length)
	// Track header
	result = append(result, 'M', 'T', 'r', 'k')
	result = append(result, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	result = append(result, buf.bytes()...)
	return result
}

// buildTrack1 构建 note track。
func (w *SMFWriter) BuildTrack1(notes []engine.RawNote, ppq uint16) []byte {
	return buildTrack1(notes, ppq, 500000) // 120 BPM default
}

func buildTrack1(notes []engine.RawNote, ppq uint16, tempoUsPerQ uint32) []byte {
	var buf writeBuffer

	usPerTick := float64(tempoUsPerQ) / float64(ppq)

	type tickEvent struct {
		tick uint64
		data []byte
	}

	var events []tickEvent

	for _, n := range notes {
		startTick := uint64(float64(n.StartMs)*1000.0 / usPerTick)
		endTick := uint64(float64(n.StartMs+n.DurationMs)*1000.0 / usPerTick)

		// Note on: 0x90 | channel note velocity
		var noteOn []byte
		noteOn = append(noteOn, 0x90)
		noteOn = append(noteOn, byte(clampNote(n.Note)))
		noteOn = append(noteOn, byte(clampVelocity(int(n.Velocity))))

		events = append(events, tickEvent{tick: startTick, data: noteOn})

		// Note off: 0x80 | channel note velocity
		var noteOff []byte
		noteOff = append(noteOff, 0x80)
		noteOff = append(noteOff, byte(clampNote(n.Note)))
		noteOff = append(noteOff, byte(64))
		events = append(events, tickEvent{tick: endTick, data: noteOff})
	}

	// 按 tick 排序
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].tick < events[i].tick {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	var lastTick uint64
	for _, ev := range events {
		delta := ev.tick - lastTick
		buf.writeVarLen(delta)
		buf.raw = append(buf.raw, ev.data...)
		lastTick = ev.tick
	}

	// End of track
	buf.writeVarLen(0)
	buf.writeByte(0xFF)
	buf.writeByte(0x2F)
	buf.writeByte(0x00)

	length := buf.len()
	result := make([]byte, 0, 8+length)
	result = append(result, 'M', 'T', 'r', 'k')
	result = append(result, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	result = append(result, buf.bytes()...)
	return result
}

// assembleSMF 组装完整的 SMF 格式 1 文件。
func assembleSMF(track0 []byte, track1 []byte, ppq uint16) []byte {
	var buf []byte

	// Header: MThd 00 00 00 06 00 01 00 02 ppq
	buf = append(buf, 'M', 'T', 'h', 'd')
	buf = append(buf, 0, 0, 0, 6) // chunk length
	buf = append(buf, 0, 1)       // format 1
	buf = append(buf, 0, 2)       // 2 tracks
	buf = append(buf, byte(ppq>>8), byte(ppq))
	buf = append(buf, track0...)
	buf = append(buf, track1...)

	return buf
}

// ===== Utility types =====

type writeBuffer struct {
	raw []byte
}

func (b *writeBuffer) writeByte(v byte) {
	b.raw = append(b.raw, v)
}

func (b *writeBuffer) writeVarLen(v uint64) {
	if v == 0 {
		b.raw = append(b.raw, 0)
		return
	}
	var tmp []byte
	for v > 0 {
		tmp = append([]byte{byte(v & 0x7F)}, tmp...)
		v >>= 7
	}
	for i := 0; i < len(tmp)-1; i++ {
		tmp[i] |= 0x80
	}
	b.raw = append(b.raw, tmp...)
}

func (b *writeBuffer) len() int  { return len(b.raw) }
func (b *writeBuffer) bytes() []byte { return b.raw }

func clampNote(n int) int {
	if n < 0 {
		return 0
	}
	if n > 127 {
		return 127
	}
	return n
}

func clampVelocity(v int) int {
	if v < 1 {
		return 1
	}
	if v > 127 {
		return 127
	}
	return v
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// ===== MIDI Importer =====

// MidiImporter 将生成的 MIDI 导入现有 MidiProject 链路。
type MidiImporter struct {
	holder *storage.Holder
}

// NewMidiImporter 创建 MIDI 导入器。
func NewMidiImporter(holder *storage.Holder) *MidiImporter {
	return &MidiImporter{holder: holder}
}

// Import 读取 MIDI 文件并导入为 MidiProject。
func (m *MidiImporter) Import(midiPath string, displayName string) (projectID uint, noteCount int, durationMs int64, fileHash string, err error) {
	store := m.holder.Current().Store

	// 使用 midi 包的 SMF 解析能力生成真实 MidiEvent
	score, parseErr := midi.ParseScoreFromPath(midiPath)
	if parseErr != nil {
		// 解析失败则回退到瘦项目记录（至少保留项目元数据）
		data, readErr := os.ReadFile(midiPath)
		if readErr != nil {
			return 0, 0, 0, "", fmt.Errorf("midi.import_failed: %w", readErr)
		}
		now := time.Now().UnixMilli()
		project := storage.MidiProject{
			DisplayName:   displayName,
			FileName:      displayName + ".mid",
			FileHash:      fmt.Sprintf("%x", simpleHash(data)),
			PPQ:           480,
			TrackCount:    1,
			ChannelCount:  1,
			DurationMs:    0,
			NoteCount:     0,
			FileSizeBytes: int64(len(data)),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		imported, err := store.ImportProject(storage.ProjectImportData{
			Project: project,
			Events:  nil,
		})
		if err != nil {
			return 0, 0, 0, "", fmt.Errorf("midi.import_failed: %w", err)
		}
		return imported.ID, 0, 0, project.FileHash, nil
	}

	data, err := os.ReadFile(midiPath)
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("midi.import_failed: %w", err)
	}
	midiHash := fmt.Sprintf("%x", sha256Hash(data))

	now := time.Now().UnixMilli()
	project := storage.MidiProject{
		DisplayName:   displayName,
		FileName:      displayName + ".mid",
		FileHash:      midiHash,
		PPQ:           score.PPQ,
		TrackCount:    score.TrackCount,
		ChannelCount:  score.ChannelCount,
		DurationMs:    score.DurationMs,
		NoteCount:     len(score.Events),
		FileSizeBytes: int64(len(data)),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	imported, err := store.ImportProject(storage.ProjectImportData{
		Project: project,
		Events:  score.Events,
	})
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("midi.import_failed: %w", err)
	}

	return imported.ID, len(score.Events), score.DurationMs, midiHash, nil
}

func simpleHash(data []byte) string {
	h := 0
	for _, b := range data {
		h = h*31 + int(b)
	}
	return fmt.Sprintf("%016x", h)
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
