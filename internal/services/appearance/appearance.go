// Package appearance provides UI appearance helpers that need native file access.
package appearance

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"YyslsPlayer/internal/utils/filex"
)

const maxBackgroundImageBytes int64 = 8 << 20

var allowedImageMIMEs = map[string]bool{
	"image/gif":  true,
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

var allowedImageExtensions = map[string]bool{
	".gif":  true,
	".jpeg": true,
	".jpg":  true,
	".png":  true,
	".webp": true,
}

type Service struct{}

func New() *Service {
	return &Service{}
}

type BackgroundImage struct {
	DataURL string `json:"dataUrl"`
	Mime    string `json:"mime"`
	Size    int    `json:"size"`
	Name    string `json:"name"`
}

func (s *Service) ImportBackgroundImage(_ context.Context, path string) (BackgroundImage, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return BackgroundImage{}, errors.New("image path required")
	}
	if !allowedImageExtensions[strings.ToLower(filepath.Ext(path))] {
		return BackgroundImage{}, errors.New("unsupported image extension")
	}

	data, err := filex.ReadLimit(path, maxBackgroundImageBytes)
	if err != nil {
		return BackgroundImage{}, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 {
		return BackgroundImage{}, errors.New("image file is empty")
	}

	mime := http.DetectContentType(data)
	if !allowedImageMIMEs[mime] {
		return BackgroundImage{}, fmt.Errorf("unsupported image type: %s", mime)
	}

	dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	return BackgroundImage{
		DataURL: dataURL,
		Mime:    mime,
		Size:    len(data),
		Name:    filepath.Base(path),
	}, nil
}
