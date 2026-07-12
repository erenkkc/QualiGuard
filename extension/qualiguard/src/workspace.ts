import * as path from "path";
import * as vscode from "vscode";
import { isSupportedFile, QualiGuardClient, QualiGuardReport } from "./client";
import { setDocumentIssues } from "./issues";
import { reportToDiagnostics } from "./diagnostics";

const SCAN_GLOB = "**/*.{py,js,jsx,ts,tsx,go,java,cs}";
const SCAN_EXCLUDE =
  "**/{node_modules,.git,dist,build,out,__pycache__,.venv,venv,.qualiguard,bin}/**";

export async function scanWorkspace(
  client: QualiGuardClient,
  collection: vscode.DiagnosticCollection,
  onProgress?: (done: number, total: number, file: string) => void,
): Promise<{ files: number; issues: number; gateWorst: string }> {
  const files = await vscode.workspace.findFiles(SCAN_GLOB, SCAN_EXCLUDE, 500);
  const supported = files.filter(f => isSupportedFile(f.fsPath));

  if (!supported.length) {
    throw new Error("Workspace'te taranacak desteklenen dosya bulunamadı.");
  }

  let totalIssues = 0;
  let gateWorst = "PASS";
  const gateRank = (g: string) => (g === "FAIL" ? 2 : g === "WARN" ? 1 : 0);

  for (let i = 0; i < supported.length; i++) {
    const uri = supported[i];
    const name = path.basename(uri.fsPath);
    onProgress?.(i + 1, supported.length, name);

    let doc: vscode.TextDocument;
    try {
      doc = await vscode.workspace.openTextDocument(uri);
    } catch {
      continue;
    }

    try {
      const report = await client.analyze(name, doc.getText());
      applyReport(uri, report, collection);
      totalIssues += report.issues?.length || 0;
      const gate = report.gate?.status || "PASS";
      if (gateRank(gate) > gateRank(gateWorst)) {
        gateWorst = gate;
      }
    } catch {
      collection.set(uri, []);
      setDocumentIssues(uri, []);
    }
  }

  return { files: supported.length, issues: totalIssues, gateWorst };
}

function applyReport(
  uri: vscode.Uri,
  report: QualiGuardReport,
  collection: vscode.DiagnosticCollection,
): void {
  const issues = report.issues || [];
  setDocumentIssues(uri, issues);
  collection.set(uri, reportToDiagnostics(issues));
}
