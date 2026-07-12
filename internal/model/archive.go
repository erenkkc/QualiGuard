package model

import "encoding/json"

const ArchiveManifestType = "archive"

type ArchiveSource struct {
	Type    string       `json:"type"`
	ZipName string       `json:"zip_name"`
	Files   []FileSource `json:"files"`
}

func (a *ArchiveSource) ManifestJSON() (string, error) {
	if a == nil {
		return "", nil
	}
	a.Type = ArchiveManifestType
	data, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseArchiveManifest(raw string) (*ArchiveSource, bool) {
	if raw == "" {
		return nil, false
	}
	var arch ArchiveSource
	if err := json.Unmarshal([]byte(raw), &arch); err != nil {
		return nil, false
	}
	if arch.Type != ArchiveManifestType || len(arch.Files) == 0 {
		return nil, false
	}
	return &arch, true
}
