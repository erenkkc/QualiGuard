package scanner

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func extractZipSafe(data []byte, dest string) (int, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("geçersiz zip: %w", err)
	}

	type entry struct {
		zip   *zip.File
		path  string
		isDir bool
	}

	var entries []entry
	for _, f := range reader.File {
		if shouldSkipZipEntry(f.Name) {
			continue
		}
		path, ok := normalizeZipEntry(f.Name)
		if !ok {
			continue
		}
		isDir := f.FileInfo().IsDir() || strings.HasSuffix(strings.TrimSpace(f.Name), "/")
		entries = append(entries, entry{zip: f, path: path, isDir: isDir})
	}

	sort.Slice(entries, func(i, j int) bool {
		depthI := strings.Count(entries[i].path, string(os.PathSeparator))
		depthJ := strings.Count(entries[j].path, string(os.PathSeparator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return entries[i].path < entries[j].path
	})

	count := 0
	for _, e := range entries {
		target := filepath.Join(dest, e.path)
		if !isPathInside(dest, target) {
			continue
		}

		if e.isDir {
			if err := ensureDirPath(target); err != nil {
				return count, err
			}
			continue
		}

		if err := ensureDirPath(filepath.Dir(target)); err != nil {
			return count, err
		}

		rc, err := e.zip.Open()
		if err != nil {
			return count, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return count, err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func shouldSkipZipEntry(name string) bool {
	n := strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
	n = strings.TrimPrefix(n, "/")
	switch {
	case n == "", n == ".", strings.HasPrefix(n, "__macosx/"), strings.HasSuffix(n, ".ds_store"):
		return true
	case strings.Contains(n, "/__macosx/"), strings.HasSuffix(n, "/thumbs.db"):
		return true
	default:
		return false
	}
}

func normalizeZipEntry(name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") {
		return "", false
	}

	parts := strings.Split(name, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = sanitizeZipPathSegment(part)
		if part == "" || part == "." {
			continue
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return "", false
	}
	return filepath.Join(clean...), true
}

func sanitizeZipPathSegment(seg string) string {
	seg = strings.TrimSpace(seg)
	seg = strings.TrimRight(seg, ". ")
	if seg == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	seg = replacer.Replace(seg)
	if isWindowsReservedName(seg) {
		seg = "_" + seg
	}
	return seg
}

func isWindowsReservedName(name string) bool {
	upper := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch upper {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func isPathInside(base, target string) bool {
	baseAbs, err1 := filepath.Abs(base)
	targetAbs, err2 := filepath.Abs(target)
	if err1 != nil || err2 != nil {
		rel, err := filepath.Rel(base, target)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
	}
	baseAbs = filepath.Clean(baseAbs)
	targetAbs = filepath.Clean(targetAbs)
	if baseAbs == targetAbs {
		return true
	}
	return strings.HasPrefix(targetAbs, baseAbs+string(os.PathSeparator))
}

// ensureDirPath creates missing directories and fixes zip entries where a file blocks a folder.
func ensureDirPath(dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return nil
	}

	parent := filepath.Dir(dir)
	if parent != dir {
		if err := ensureDirPath(parent); err != nil {
			return err
		}
	}

	if info, err := os.Lstat(dir); err == nil {
		if info.IsDir() {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("klasör oluşturulamadı (dosya engelliyor): %s", dir)
		}
	}

	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}
