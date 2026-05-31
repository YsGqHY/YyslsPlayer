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
