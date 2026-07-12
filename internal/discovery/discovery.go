package discovery

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var languageExtensions = map[string]string{
	".py":   "python",
	".pyw":  "python",
	".go":   "go",
	".js":   "javascript",
	".jsx":  "javascript",
	".ts":   "javascript",
	".tsx":  "javascript",
	".mjs":  "javascript",
	".cjs":  "javascript",
	".java": "java",
	".cs":   "csharp",
}

func Discover(root string, languages []string, exclusions, inclusions []string) ([]string, error) {
	allowed := make(map[string]struct{}, len(languages))
	for _, lang := range languages {
		allowed[strings.ToLower(lang)] = struct{}{}
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if matchesAny(path, exclusions) {
				return filepath.SkipDir
			}
			return nil
		}

		if len(inclusions) > 0 && !matchesAny(path, inclusions) {
			return nil
		}
		if matchesAny(path, exclusions) {
			return nil
		}

		lang := LanguageForPath(path)
		if lang == "" {
			return nil
		}
		if _, ok := allowed[lang]; !ok {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func LanguageForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return languageExtensions[ext]
}

func MatchesAny(path string, patterns []string) bool {
	return matchesAny(path, patterns)
}

func matchesAny(path string, patterns []string) bool {
	clean := filepath.ToSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if ok, _ := filepath.Match(pattern, clean); ok {
			return true
		}
		if ok, _ := filepath.Match(pattern, filepath.Base(clean)); ok {
			return true
		}
		if strings.Contains(pattern, "**") {
			if matchGlobstar(clean, pattern) {
				return true
			}
		}
	}
	return false
}

func matchGlobstar(path, pattern string) bool {
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return false
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	if prefix != "" && !strings.HasPrefix(path, prefix) {
		return false
	}
	if suffix == "" || suffix == "*" {
		return true
	}
	if strings.HasSuffix(suffix, "/**") {
		dir := strings.TrimSuffix(suffix, "/**")
		return strings.Contains(path, "/"+dir+"/") || strings.HasPrefix(path, dir+"/")
	}
	if strings.HasPrefix(suffix, "*.") {
		return strings.HasSuffix(path, suffix[1:])
	}
	return strings.Contains(path, suffix)
}
