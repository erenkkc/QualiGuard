package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/qualiguard/qualiguard/internal/fingerprint"
	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/model"
	"github.com/qualiguard/qualiguard/internal/store"
)

type Processor struct {
	store *store.Store
}

func New(s *store.Store) *Processor {
	return &Processor{store: s}
}

func (p *Processor) ProcessReport(ctx context.Context, report *model.Report) (*model.UploadResult, error) {
	if report.Project.Key == "" {
		return nil, fmt.Errorf("project.key is required")
	}

	project, err := p.store.EnsureProject(ctx, report.Project.Key, report.Project.Name)
	if err != nil {
		return nil, err
	}

	analysisID, err := p.store.CreateAnalysis(ctx, project.ID, report, "")
	if err != nil {
		return nil, err
	}

	if report.Source != nil && strings.TrimSpace(report.Source.Text) != "" {
		if err := p.store.SaveAnalysisSource(ctx, analysisID, report.Source.Filename, report.Source.Text); err != nil {
			return nil, err
		}
	}
	if report.Archive != nil && len(report.Archive.Files) > 0 {
		manifest, err := report.Archive.ManifestJSON()
		if err != nil {
			return nil, err
		}
		name := report.Archive.ZipName
		if name == "" {
			name = "archive.zip"
		}
		if err := p.store.SaveAnalysisSource(ctx, analysisID, name, manifest); err != nil {
			return nil, err
		}
	}

	if err := p.store.SaveMeasures(ctx, analysisID, report.Measures); err != nil {
		return nil, err
	}

	openFPs, err := p.store.OpenIssueFingerprints(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	fpIndex, err := p.store.IssueFingerprintIndex(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(report.Issues))
	newCount := 0

	for _, issue := range report.Issues {
		fingerprint.Annotate(&issue)
		if issue.Fingerprint == "" {
			continue
		}
		seen[issue.Fingerprint] = struct{}{}

		ref, known := fpIndex[issue.Fingerprint]
		if known && ref.Status == model.StatusClosed && store.IsSuppressedResolution(ref.Resolution) {
			continue
		}

		existingID := ""
		if known {
			existingID = ref.ID
		} else if id := openFPs[issue.Fingerprint]; id != "" {
			existingID = id
		}
		if existingID == "" {
			newCount++
		}
		if err := p.store.UpsertIssue(ctx, project.ID, analysisID, issue, existingID); err != nil {
			return nil, err
		}
	}

	closedCount, err := p.store.CloseMissingIssues(ctx, project.ID, seen)
	if err != nil {
		return nil, err
	}

	openCount, err := p.store.CountOpenIssues(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	gateInput, err := p.store.OpenGateInput(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	gateResult := gate.Evaluate(gateInput)
	if err := p.store.UpdateAnalysisSummary(
		ctx, analysisID, string(gateResult.Status),
		len(report.Issues), newCount, openCount, closedCount,
	); err != nil {
		return nil, err
	}

	return &model.UploadResult{
		AnalysisID:   analysisID,
		ProjectKey:   project.Key,
		IssuesFound:  len(report.Issues),
		IssuesNew:    newCount,
		IssuesOpen:   openCount,
		IssuesClosed: closedCount,
		Gate:         gate.ToModelResult(gateResult),
	}, nil
}
