package reporter

import (
	"os"

	"github.com/qualiguard/qualiguard/internal/github"
	"github.com/qualiguard/qualiguard/internal/model"
)

func WritePRComment(outputPath string, report *model.Report) error {
	body := github.BuildPRComment(report)
	if outputPath == "" || outputPath == "-" {
		_, err := os.Stdout.WriteString(body)
		return err
	}
	return os.WriteFile(outputPath, []byte(body), 0o644)
}
