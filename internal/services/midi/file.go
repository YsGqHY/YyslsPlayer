package midi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxMidiFileSizeBytes int64 = 5 * 1024 * 1024
)

var (
	ErrMidiFileNotFound      = errors.New("MIDI_FILE_NOT_FOUND")
	ErrMidiUnsupportedFormat = errors.New("MIDI_UNSUPPORTED_FORMAT")
	ErrMidiTooLarge          = errors.New("MIDI_TOO_LARGE")
)

type midiFileData struct {
	Path     string
	Name     string
	Size     int64
	Bytes    []byte
	SHA256   string
	FileHash string
}

func readMidiFile(path string) (midiFileData, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return midiFileData{}, fmt.Errorf("%w: path required", ErrMidiFileNotFound)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return midiFileData{}, fmt.Errorf("%w: %v", ErrMidiFileNotFound, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return midiFileData{}, fmt.Errorf("%w: %s", ErrMidiFileNotFound, abs)
		}
		return midiFileData{}, err
	}
	if info.IsDir() {
		return midiFileData{}, fmt.Errorf("%w: %s is a directory", ErrMidiUnsupportedFormat, abs)
	}
	if !isMidiExtension(info.Name()) {
		return midiFileData{}, fmt.Errorf("%w: %s", ErrMidiUnsupportedFormat, info.Name())
	}
	if info.Size() > maxMidiFileSizeBytes {
		return midiFileData{}, fmt.Errorf("%w: %d > %d", ErrMidiTooLarge, info.Size(), maxMidiFileSizeBytes)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return midiFileData{}, err
	}
	hash := sha256.Sum256(data)
	hexHash := hex.EncodeToString(hash[:])

	return midiFileData{
		Path:     abs,
		Name:     filepath.Base(abs),
		Size:     info.Size(),
		Bytes:    data,
		SHA256:   hexHash,
		FileHash: "sha256:" + hexHash,
	}, nil
}

func findMidiFilesInDirectory(dir string) ([]string, error) {
	cleaned := strings.TrimSpace(dir)
	if cleaned == "" {
		return nil, fmt.Errorf("%w: directory required", ErrMidiFileNotFound)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMidiFileNotFound, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrMidiFileNotFound, abs)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s is not a directory", ErrMidiUnsupportedFormat, abs)
	}

	paths := make([]string, 0)
	if err := filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if isMidiExtension(entry.Name()) {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func isMidiExtension(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mid", ".midi":
		return true
	default:
		return false
	}
}

// expandDroppedPaths 把拖入的原始路径集合展开为可导入的 MIDI 文件路径列表：
//   - 目录递归展开为其中的 MIDI 文件
//   - 普通 MIDI 文件原样保留
//   - 非 MIDI 文件原样保留（交由后续 readMidiFile 报具体失败原因，便于前端逐项反馈）
//
// 返回结果去重并保持去重后的稳定顺序。空白路径会被忽略。
func expandDroppedPaths(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	appendPath := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}

	for _, raw := range paths {
		cleaned := strings.TrimSpace(raw)
		if cleaned == "" {
			continue
		}
		info, err := os.Stat(cleaned)
		if err != nil {
			// 路径无法访问：原样保留，让导入阶段给出明确失败项。
			appendPath(cleaned)
			continue
		}
		if info.IsDir() {
			dirFiles, err := findMidiFilesInDirectory(cleaned)
			if err != nil {
				return nil, err
			}
			for _, f := range dirFiles {
				appendPath(f)
			}
			continue
		}
		appendPath(cleaned)
	}
	return out, nil
}
