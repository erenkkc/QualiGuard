import * as path from "path";
import * as vscode from "vscode";
import {
  isSupportedFile,
  QualiGuardClient,
  QualiGuardReport,
} from "./client";
import { reportToDiagnostics } from "./diagnostics";
import { clearAllIssues, registerHoverProvider, setDocumentIssues } from "./issues";
import { scanWorkspace } from "./workspace";

const DIAG_COLLECTION = "qualiguard";

let client: QualiGuardClient | undefined;
let statusItem: vscode.StatusBarItem;
let scanning = false;

export function activate(context: vscode.ExtensionContext) {
  const collection = vscode.languages.createDiagnosticCollection(DIAG_COLLECTION);
  context.subscriptions.push(collection);

  statusItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
  statusItem.command = "qualiguard.connect";
  context.subscriptions.push(statusItem);

  registerHoverProvider(context);
  client = buildClient();
  updateStatus("QualiGuard", "Sunucuya bağlanmak için tıkla");

  context.subscriptions.push(
    vscode.commands.registerCommand("qualiguard.connect", () => connect(context)),
    vscode.commands.registerCommand("qualiguard.scanFile", () => scanActiveEditor(collection)),
    vscode.commands.registerCommand("qualiguard.scanWorkspace", () => runWorkspaceScan(collection)),
    vscode.commands.registerCommand("qualiguard.clearDiagnostics", () => {
      collection.clear();
      clearAllIssues();
      updateStatus("QualiGuard", "Uyarılar temizlendi");
    }),
    vscode.workspace.onDidSaveTextDocument(doc => {
      if (getConfig().scanOnSave) {
        void scanDocument(doc, collection);
      }
    }),
    vscode.window.onDidChangeActiveTextEditor(editor => {
      if (editor && getConfig().scanOnOpen) {
        void scanDocument(editor.document, collection);
      }
    }),
    vscode.workspace.onDidChangeConfiguration(e => {
      if (e.affectsConfiguration("qualiguard")) {
        client = buildClient();
        const saved = context.globalState.get<string>("qualiguard.token");
        if (saved) {
          client?.setToken(saved);
        }
      }
    }),
  );

  void connect(context, true);

  const active = vscode.window.activeTextEditor;
  if (active && getConfig().scanOnOpen) {
    void scanDocument(active.document, collection);
  }
}

export function deactivate() {
  statusItem?.dispose();
}

function getConfig() {
  const cfg = vscode.workspace.getConfiguration("qualiguard");
  return {
    serverUrl: cfg.get<string>("serverUrl", "http://127.0.0.1:9000"),
    token: cfg.get<string>("token", ""),
    scanOnSave: cfg.get<boolean>("scanOnSave", true),
    scanOnOpen: cfg.get<boolean>("scanOnOpen", true),
  };
}

function buildClient(): QualiGuardClient {
  const { serverUrl, token } = getConfig();
  return new QualiGuardClient(serverUrl, token);
}

async function connect(context: vscode.ExtensionContext, silent = false) {
  if (!client) {
    client = buildClient();
  }
  const { token: cfgToken } = getConfig();
  if (cfgToken?.startsWith("qg_")) {
    client.setToken(cfgToken);
  } else {
    const saved = context.globalState.get<string>("qualiguard.token");
    if (saved) {
      client.setToken(saved);
    }
  }

  try {
    const ok = await client.health();
    if (!ok) {
      throw new Error("Sunucu yanıt vermiyor — server.bat açık mı?");
    }

    let token: string;
    try {
      token = await client.ensureToken();
    } catch (bootstrapErr) {
      const cfg = await client.publicConfig();
      if (cfg.auth_required) {
        const pw = await vscode.window.showInputBox({
          title: "QualiGuard panel şifresi",
          prompt: "Uzak sunucuya bağlanmak için panel şifrenizi girin",
          password: true,
          ignoreFocusOut: true,
        });
        if (!pw) {
          throw bootstrapErr;
        }
        token = await client.loginWithPassword(pw);
      } else {
        throw bootstrapErr;
      }
    }

    await context.globalState.update("qualiguard.token", token);
    updateStatus("QualiGuard ✓", "Bağlı — kaydedince veya workspace taraması");
    if (!silent) {
      vscode.window.showInformationMessage("QualiGuard sunucusuna bağlandı.");
    }
  } catch (err) {
    updateStatus("QualiGuard ✗", String(err));
    if (!silent) {
      vscode.window.showErrorMessage(`QualiGuard bağlantı hatası: ${err}`);
    }
  }
}

async function scanActiveEditor(collection: vscode.DiagnosticCollection) {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("Açık dosya yok.");
    return;
  }
  await scanDocument(editor.document, collection);
}

async function runWorkspaceScan(collection: vscode.DiagnosticCollection) {
  if (!client) {
    client = buildClient();
  }
  if (scanning) {
    vscode.window.showWarningMessage("Tarama zaten devam ediyor.");
    return;
  }

  scanning = true;
  try {
    await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: "QualiGuard workspace taraması",
        cancellable: false,
      },
      async progress => {
        let last = 0;
        const result = await scanWorkspace(client!, collection, (done, total, file) => {
          progress.report({
            increment: done > last ? done - last : 0,
            message: `${done}/${total} · ${file}`,
          });
          last = done;
          updateStatus(`QualiGuard … ${done}/${total}`, file);
        });

        const gate = result.gateWorst === "FAIL" ? "KALIR" : result.gateWorst === "WARN" ? "UYARI" : "GEÇER";
        updateStatus(
          `QualiGuard: ${result.issues} uyarı`,
          `${result.files} dosya · Kapı: ${gate}`,
        );
        vscode.window.showInformationMessage(
          `QualiGuard: ${result.files} dosya, ${result.issues} uyarı (kapı: ${gate})`,
        );
      },
    );
  } catch (err) {
    updateStatus("QualiGuard ✗", String(err));
    vscode.window.showErrorMessage(`Workspace taraması: ${err}`);
  } finally {
    scanning = false;
  }
}

async function scanDocument(
  document: vscode.TextDocument,
  collection: vscode.DiagnosticCollection,
) {
  if (document.uri.scheme !== "file") {
    return;
  }
  if (!isSupportedFile(document.fileName)) {
    return;
  }
  if (!client) {
    client = buildClient();
  }
  if (scanning) {
    return;
  }
  scanning = true;
  updateStatus("QualiGuard …", "Taranıyor");

  try {
    const report = await client.analyze(
      path.basename(document.fileName),
      document.getText(),
    );
    applyReport(document.uri, report, collection);
    const n = report.issues?.length || 0;
    const gate = report.gate?.status_tr || report.gate?.status || "";
    updateStatus(
      n ? `QualiGuard: ${n} uyarı` : "QualiGuard ✓",
      gate ? `Kapı: ${gate}` : "Temiz",
    );
  } catch (err) {
    updateStatus("QualiGuard ✗", String(err));
    vscode.window.showErrorMessage(`QualiGuard tarama hatası: ${err}`);
  } finally {
    scanning = false;
  }
}

function applyReport(
  uri: vscode.Uri,
  report: QualiGuardReport,
  collection: vscode.DiagnosticCollection,
) {
  const issues = report.issues || [];
  setDocumentIssues(uri, issues);
  collection.set(uri, reportToDiagnostics(issues));
}

function updateStatus(text: string, tooltip: string) {
  statusItem.text = text;
  statusItem.tooltip = tooltip;
  statusItem.show();
}
