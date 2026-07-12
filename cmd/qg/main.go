package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/qualiguard/qualiguard/internal/baseline"
	"github.com/qualiguard/qualiguard/internal/client"
	"github.com/qualiguard/qualiguard/internal/config"
	"github.com/qualiguard/qualiguard/internal/gate"
	"github.com/qualiguard/qualiguard/internal/metrics"
	"github.com/qualiguard/qualiguard/internal/reporter"
	"github.com/qualiguard/qualiguard/internal/scanner"
	"github.com/spf13/cobra"
)

const version = "1.0.0"

func main() {
	root := &cobra.Command{
		Use:     "qg",
		Short:   "QualiGuard — AI-ready static code analysis",
		Version: version,
	}

	root.AddCommand(newScanCommand())

	if err := root.Execute(); err != nil {
		os.Exit(2)
	}
}

func newScanCommand() *cobra.Command {
	var (
		configPath  string
		sources     []string
		projectKey  string
		projectName string
		outputPath  string
		format      string
		prComment   string
		verbose     bool
		upload      bool
		serverURL   string
		token       string
		failOnGate   bool
		baselinePath string
		saveBaseline string
		noAI         bool
		incremental  bool
		baseRef      string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Analyze source code and produce a report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			if projectKey != "" {
				cfg.Project.Key = projectKey
			}
			if projectName != "" {
				cfg.Project.Name = projectName
			}
			if len(sources) > 0 {
				cfg.Sources = sources
			}

			workDir, err := os.Getwd()
			if err != nil {
				return err
			}

			s := scanner.New(cfg, workDir, version)
			report, err := s.Scan(cmd.Context(), scanner.ScanOptions{
				Incremental: incremental,
				BaseRef:     baseRef,
			})
			if err != nil {
				return err
			}

			if baselinePath != "" {
				bl, err := baseline.Load(baselinePath)
				if err != nil {
					return err
				}
				before := len(report.Issues)
				report.Issues = bl.Filter(report.Issues)
				if verbose {
					fmt.Fprintf(os.Stderr, "Baseline: %d sorun yok sayıldı\n", before-len(report.Issues))
				}
				report.Measures = metrics.FromIssues(
					report.Issues,
					report.Measures.Files,
					report.Measures.Ncloc,
					report.Measures.Complexity,
				)
			}

			if saveBaseline != "" {
				if err := baseline.Save(saveBaseline, report.Issues); err != nil {
					return err
				}
				if verbose {
					fmt.Fprintf(os.Stderr, "Baseline kaydedildi: %s\n", saveBaseline)
				}
			}

			if noAI {
				for i := range report.Issues {
					report.Issues[i].AIExplanation = nil
				}
			}

			if verbose {
				if incremental {
					fmt.Fprintf(os.Stderr, "Incremental scan (base: %s): %d dosya\n",
						baseRefOrDefault(baseRef), report.Analysis.ScannedFiles)
				}
				fmt.Fprintf(os.Stderr, "Scanned %d files, found %d issues\n",
					report.Measures.Files, len(report.Issues))
			}

			gateResult := gate.Evaluate(gate.InputFromIssues(report.Issues))
			if report.Gate == nil {
				modelResult := gate.ToModelResult(gateResult)
				report.Gate = &modelResult
			}
			printGateSummary(gateResult)

			if upload {
				if serverURL == "" {
					return fmt.Errorf("--server-url is required with --upload")
				}
				if token == "" {
					return fmt.Errorf("--token is required with --upload")
				}
				c := client.New(serverURL, token)
				if err := c.Health(cmd.Context()); err != nil {
					return fmt.Errorf("server not reachable: %w", err)
				}
				result, err := c.UploadReport(cmd.Context(), report)
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Uploaded analysis %s — new: %d, open: %d, closed: %d — kalite kapısı: %s\n",
					result.AnalysisID, result.IssuesNew, result.IssuesOpen, result.IssuesClosed, result.Gate.StatusTR)
			}

			writeReport := !(upload && (outputPath == "" || outputPath == "-"))
			if writeReport {
				switch format {
				case "json":
					if err := reporter.WriteJSON(outputPath, report); err != nil {
						return err
					}
				case "sarif":
					if err := reporter.WriteSARIF(outputPath, report); err != nil {
						return err
					}
				default:
					return fmt.Errorf("unsupported format: %s (use json or sarif)", format)
				}

				if outputPath != "" && outputPath != "-" && verbose {
					fmt.Fprintf(os.Stderr, "Report written to %s\n", outputPath)
				}
			}

			if prComment != "" {
				if err := reporter.WritePRComment(prComment, report); err != nil {
					return err
				}
				if verbose {
					fmt.Fprintf(os.Stderr, "PR yorumu yazıldı: %s\n", prComment)
				}
			}

			if failOnGate && gateResult.Status == gate.StatusFail {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to qualiguard.yaml")
	cmd.Flags().StringArrayVar(&sources, "sources", nil, "Source directories to scan")
	cmd.Flags().StringVar(&projectKey, "project-key", "", "Project key override")
	cmd.Flags().StringVar(&projectName, "project-name", "", "Project name override")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "-", "Output file path (- for stdout)")
	cmd.Flags().StringVarP(&format, "format", "f", "json", "Report format: json | sarif")
	cmd.Flags().StringVar(&prComment, "pr-comment", "", "Write GitHub PR summary markdown to this file")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging to stderr")
	cmd.Flags().BoolVar(&upload, "upload", false, "Upload report to QualiGuard server")
	cmd.Flags().StringVar(&serverURL, "server-url", "http://127.0.0.1:9000", "QualiGuard server URL")
	cmd.Flags().StringVar(&token, "token", "", "API token for server upload")
	cmd.Flags().BoolVar(&failOnGate, "fail-on-gate", false, "Exit with code 1 when quality gate fails")
	cmd.Flags().StringVar(&baselinePath, "baseline", "", "Ignore issues matching baseline fingerprints JSON")
	cmd.Flags().StringVar(&saveBaseline, "save-baseline", "", "Save current issue fingerprints as baseline")
	cmd.Flags().BoolVar(&noAI, "no-ai", false, "Disable AI explanations in report")
	cmd.Flags().BoolVar(&incremental, "incremental", false, "Scan only files changed since base ref (requires git)")
	cmd.Flags().StringVar(&baseRef, "base", "main", "Base git ref for incremental scan (e.g. main, origin/main)")

	return cmd
}

func baseRefOrDefault(ref string) string {
	if strings.TrimSpace(ref) == "" {
		return "main"
	}
	return ref
}

func printGateSummary(result gate.Result) {
	fmt.Fprintf(os.Stderr, "Kalite Kapısı (%s): %s\n", result.NameTR, result.StatusTR)
	for _, cond := range result.Conditions {
		mark := "✓"
		if !cond.Passed {
			mark = "✗"
		}
		fmt.Fprintf(os.Stderr, "  %s %s: %.0f (eşik: %.0f)\n", mark, cond.LabelTR, cond.Actual, cond.Threshold)
	}
}
