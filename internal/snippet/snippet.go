package snippet

import (
	"fmt"
	"os"
	"strings"
)

func FromFile(path string, line, contextLines int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if line <= 0 || line > len(lines) {
		return "", nil
	}

	start := line - contextLines - 1
	if start < 0 {
		start = 0
	}
	end := line + contextLines
	if end > len(lines) {
		end = len(lines)
	}

	var out strings.Builder
	for i := start; i < end; i++ {
		prefix := "  "
		if i+1 == line {
			prefix = "> "
		}
		out.WriteString(fmt.Sprintf("%s%4d | %s\n", prefix, i+1, lines[i]))
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func LineAt(path string, line int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if line <= 0 || line > len(lines) {
		return "", nil
	}
	return lines[line-1], nil
}
