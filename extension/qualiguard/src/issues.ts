import * as vscode from "vscode";
import { QualiGuardIssue } from "./client";

const issuesByUri = new Map<string, QualiGuardIssue[]>();

export function setDocumentIssues(uri: vscode.Uri, issues: QualiGuardIssue[]): void {
  issuesByUri.set(uri.toString(), issues);
}

export function getDocumentIssues(uri: vscode.Uri): QualiGuardIssue[] {
  return issuesByUri.get(uri.toString()) || [];
}

export function clearAllIssues(): void {
  issuesByUri.clear();
}

export function registerHoverProvider(context: vscode.ExtensionContext): void {
  const selector: vscode.DocumentSelector = [
    { scheme: "file", language: "python" },
    { scheme: "file", language: "javascript" },
    { scheme: "file", language: "typescript" },
    { scheme: "file", language: "go" },
    { scheme: "file", language: "java" },
    { scheme: "file", language: "csharp" },
  ];

  context.subscriptions.push(
    vscode.languages.registerHoverProvider(selector, {
      provideHover(document, position) {
        const lineNo = position.line + 1;
        const hits = getDocumentIssues(document.uri).filter(i => (i.line || 1) === lineNo);
        if (!hits.length) {
          return undefined;
        }

        const parts = hits.map(issue => {
          const lines = [
            `**${issue.severity}** · \`${issue.rule_key}\``,
            issue.message,
          ];
          if (issue.fix_suggestion?.trim()) {
            lines.push(`_Öneri:_ ${issue.fix_suggestion.trim()}`);
          }
          return lines.join("\n\n");
        });

        const md = new vscode.MarkdownString(parts.join("\n\n---\n\n"));
        md.isTrusted = true;
        return new vscode.Hover(md);
      },
    }),
  );
}
