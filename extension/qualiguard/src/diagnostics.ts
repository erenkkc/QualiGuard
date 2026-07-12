import * as vscode from "vscode";
import { QualiGuardIssue } from "./client";

export function issueToDiagnostic(issue: QualiGuardIssue): vscode.Diagnostic {
  const line = Math.max(0, (issue.line || 1) - 1);
  const col = Math.max(0, (issue.column || 1) - 1);
  const range = new vscode.Range(line, col, line, Math.max(col + 1, 200));

  const severity = mapSeverity(issue.severity);
  const diag = new vscode.Diagnostic(range, formatMessage(issue), severity);
  diag.source = "QualiGuard";
  diag.code = issue.rule_key;
  return diag;
}

function mapSeverity(sev: string): vscode.DiagnosticSeverity {
  switch (sev) {
    case "BLOCKER":
    case "CRITICAL":
      return vscode.DiagnosticSeverity.Error;
    case "MAJOR":
      return vscode.DiagnosticSeverity.Warning;
    default:
      return vscode.DiagnosticSeverity.Information;
  }
}

function formatMessage(issue: QualiGuardIssue): string {
  const parts = [issue.message];
  if (issue.fix_suggestion?.trim()) {
    parts.push(`Öneri: ${issue.fix_suggestion.trim()}`);
  }
  return parts.join(" — ");
}

export function reportToDiagnostics(issues: QualiGuardIssue[]): vscode.Diagnostic[] {
  return issues.map(issueToDiagnostic);
}
