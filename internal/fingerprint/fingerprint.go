package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/qualiguard/qualiguard/internal/model"
)

func Issue(ruleKey, file, message string, line int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%s", ruleKey, file, line, message)))
	return "sha256:" + hex.EncodeToString(sum[:16])
}

func Annotate(issue *model.Issue) {
	if issue.Fingerprint == "" {
		issue.Fingerprint = Issue(issue.RuleKey, issue.File, issue.Message, issue.Line)
	}
}
