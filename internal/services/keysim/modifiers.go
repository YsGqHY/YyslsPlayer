package keysim

import (
	"encoding/json"
	"fmt"
)

type modifierJSON struct {
	Label      string `json:"label"`
	VirtualKey int    `json:"virtualKey"`
	ScanCode   int    `json:"scanCode"`
}

func DecodeModifiers(raw string) ([]Key, error) {
	if raw == "" {
		return nil, nil
	}
	var rows []modifierJSON
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("decode modifier keys: %w", err)
	}
	out := make([]Key, 0, len(rows))
	for _, row := range rows {
		key := Key{Label: row.Label, VirtualKey: row.VirtualKey, ScanCode: row.ScanCode}
		if err := validateKey(key); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, nil
}
