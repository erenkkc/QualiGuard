const DEFAULT_CODE = `import os

DB_PASSWORD = "super-secret-123"

def get_user(user_id):
    unused = 42
    try:
        raise ValueError("fail")
    except:
        pass

    eval("print('hi')")
    cursor = None
    cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")
`;

const TR = {
  severity: {
    BLOCKER: "Engelleyici",
    CRITICAL: "Kritik",
    MAJOR: "Önemli",
    MINOR: "Küçük",
    INFO: "Bilgi",
  },
  type: {
    BUG: "Hata",
    VULNERABILITY: "Güvenlik Açığı",
    CODE_SMELL: "Kod Kokusu",
    SECURITY_HOTSPOT: "Güvenlik Noktası",
  },
  gate: {
    PASS: "Geçti",
    WARN: "Uyarı",
    FAIL: "Kaldı",
  },
  resolution: {
    FALSE_POSITIVE: "Yanlış alarm",
    WONTFIX: "Düzeltilmeyecek",
    FIXED: "Düzeltildi",
  },
  status: {
    OPEN: "Açık",
    CLOSED: "Kapalı",
  },
};

function trSeverity(value) {
  return TR.severity[value] || value;
}

function trType(value) {
  return TR.type[value] || value;
}

function trGate(value) {
  return TR.gate[value] || value;
}

function trResolution(value) {
  return TR.resolution[value] || value;
}

const state = {
  token: "",
  projects: [],
  issues: [],
  filter: "ALL",
  typeFilter: "ALL",
  statusFilter: "OPEN",
  selectedId: null,
  projectKey: "",
  projectGate: null,
  playground: {
    source: localStorage.getItem("qg_playground_code") || DEFAULT_CODE,
    issues: [],
    selectedLine: null,
    gate: null,
    lastReport: null,
    lastAnalyzedSource: "",
    analyzeTimer: null,
    analyzeGen: 0,
    analyzing: false,
    language: "",
  },
  uploadPreview: {
    source: "",
    issues: [],
    selectedLine: null,
    gate: null,
    filename: "",
    language: "",
    ready: false,
    archive: false,
    files: [],
    activeFile: "",
  },
  projectSource: {
    source: "",
    issues: [],
    selectedLine: null,
    filename: "",
    archive: false,
    files: [],
    activeFile: "",
  },
  editorMode: "playground",
  aiCache: {},
  aiActive: false,
  aiProvider: "",
  aiModel: "",
  chatMessages: JSON.parse(sessionStorage.getItem("qg_chat") || "[]"),
  chatBusy: false,
  selectedProjects: new Set(),
  projectSearch: "",
};

function syncAIStatus(ai) {
  state.aiActive = !!ai?.active;
  state.aiProvider = ai?.provider || "";
  state.aiModel = ai?.model || "";
}

function aiProviderLabel() {
  if (!state.aiActive) return "";
  const names = { openai: "OpenAI", gemini: "Gemini", ollama: "Ollama" };
  const name = names[state.aiProvider] || state.aiProvider || "AI";
  return state.aiModel ? `${name} (${state.aiModel})` : name;
}

function showToast(message, type = "info") {
  let el = document.getElementById("toast");
  if (!el) {
    el = document.createElement("div");
    el.id = "toast";
    document.body.appendChild(el);
  }
  el.className = `toast ${type}`;
  el.textContent = message;
  el.classList.add("show");
  clearTimeout(el._timer);
  el._timer = setTimeout(() => el.classList.remove("show"), 3500);
}

function renderMeasuresGrid(measures) {
  if (!measures) return "";
  const rows = [
    ["Dosya", measures.files],
    ["Satır (NCLOC)", measures.ncloc],
    ["Karmaşıklık", measures.complexity],
    ["Hata", measures.bugs],
    ["Güvenlik", measures.vulnerabilities],
    ["Stil", measures.code_smells],
  ].filter(([, v]) => v != null && Number(v) > 0);
  if (!rows.length) return "";
  return `<div class="measures-grid">${rows.map(([label, v]) => `
    <div class="measure-card"><div class="label">${label}</div><div class="value">${v}</div></div>
  `).join("")}</div>`;
}

async function loadDemoToPlayground(name) {
  try {
    const demo = await api(`/api/v1/demo/${encodeURIComponent(name)}`);
    state.playground.source = demo.source;
    state.playground.lastAnalyzedSource = "";
    state.playground.issues = [];
    localStorage.setItem("qg_playground_code", demo.source);
    location.hash = "#/playground";
    showToast(`${demo.filename} yüklendi`, "ok");
  } catch (err) {
    showToast(err.message, "error");
  }
}

async function rescanProject(key) {
  try {
    showToast("Proje yeniden taranıyor...");
    await api(`/api/v1/projects/${encodeURIComponent(key)}/rescan`, { method: "POST" });
    showToast("Tarama tamamlandı", "ok");
    renderProject(key);
  } catch (err) {
    showToast(err.message, "error");
  }
}

async function deleteProject(key, name) {
  if (!confirm(`"${name || key}" projesini silmek istediğine emin misin?`)) return;
  try {
    await api(`/api/v1/projects/${encodeURIComponent(key)}`, { method: "DELETE" });
    state.selectedProjects?.delete(key);
    showToast("Proje silindi", "ok");
    location.hash = "#/projects";
    renderProjects();
  } catch (err) {
    showToast(err.message, "error");
  }
}

async function bulkDeleteProjects() {
  const keys = [...(state.selectedProjects || [])];
  if (!keys.length) return;
  if (!confirm(`${keys.length} proje silinsin mi? Bu işlem geri alınamaz.`)) return;
  try {
    const result = await api("/api/v1/projects/bulk-delete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ keys }),
    });
    state.selectedProjects = new Set();
    showToast(`${result.count || keys.length} proje silindi`, "ok");
    renderProjects();
  } catch (err) {
    showToast(err.message, "error");
  }
}

function updateBulkDeleteButton() {
  const btn = document.getElementById("bulk-delete");
  const n = state.selectedProjects?.size || 0;
  if (!btn) return;
  btn.disabled = n === 0;
  btn.textContent = n ? `Seçilenleri sil (${n})` : "Seçilenleri sil";
}

async function downloadProjectExport(key, format) {
  try {
    await ensureToken();
    const res = await fetch(`/api/v1/projects/${encodeURIComponent(key)}/export?format=${format}`, {
      headers: { Authorization: "Bearer " + state.token },
    });
    if (!res.ok) throw new Error(await res.text());
    const blob = await res.blob();
    const dispo = res.headers.get("Content-Disposition") || "";
    const match = dispo.match(/filename="([^"]+)"/);
    const filename = match ? match[1] : `qualiguard-${key}.${format}`;
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
    showToast("Rapor indirildi", "ok");
  } catch (err) {
    showToast(err.message, "error");
  }
}

async function previewProjectExport(key) {
  try {
    await ensureToken();
    const res = await fetch(`/api/v1/projects/${encodeURIComponent(key)}/export?format=html`, {
      headers: { Authorization: "Bearer " + state.token },
    });
    if (!res.ok) throw new Error(await res.text());
    const html = await res.text();
    const win = window.open("", "_blank");
    if (!win) {
      showToast("Pop-up engellendi — tarayıcıda izin verin", "error");
      return;
    }
    win.document.open();
    win.document.write(html);
    win.document.close();
  } catch (err) {
    showToast(err.message, "error");
  }
}

function filterProjects(projects, query) {
  const q = (query || "").trim().toLowerCase();
  if (!q) return projects;
  return projects.filter(p =>
    (p.name || "").toLowerCase().includes(q) ||
    (p.key || "").toLowerCase().includes(q)
  );
}

function editorCtx() {
  if (state.editorMode === "upload") return state.uploadPreview;
  if (state.editorMode === "project") return state.projectSource;
  return state.playground;
}

function isValidToken(token) {
  return token && token !== "__QG_TOKEN__" && token.startsWith("qg_");
}

async function resolveToken() {
  if (isValidToken(window.__QG_TOKEN__)) return window.__QG_TOKEN__;
  const saved = localStorage.getItem("qg_token");
  if (isValidToken(saved)) return saved;
  const res = await fetch("/api/bootstrap");
  if (res.ok) return (await res.json()).token;
  const cfg = await fetch("/api/public/config").then(r => r.json()).catch(() => ({}));
  if (cfg.auth_required) {
    const next = encodeURIComponent(location.pathname + location.search + location.hash);
    window.location.href = `/login?next=${next}`;
    throw new Error("Giriş gerekli");
  }
  throw new Error("API token alınamadı.");
}

async function ensureToken() {
  if (isValidToken(state.token)) return;
  const saved = localStorage.getItem("qg_token");
  if (isValidToken(saved)) {
    state.token = saved;
    return;
  }
  if (isValidToken(window.__QG_TOKEN__)) {
    state.token = window.__QG_TOKEN__;
    localStorage.setItem("qg_token", state.token);
    return;
  }
  const res = await fetch("/api/bootstrap");
  if (res.ok) {
    const data = await res.json();
    state.token = data.token;
    syncAIStatus(data.ai);
    localStorage.setItem("qg_token", state.token);
    return;
  }
  const cfg = await fetch("/api/public/config").then(r => r.json()).catch(() => ({}));
  if (cfg.auth_required) {
    const next = encodeURIComponent(location.pathname + location.search + location.hash);
    window.location.href = `/login?next=${next}`;
    throw new Error("Giriş gerekli");
  }
  throw new Error("API token alınamadı.");
}

async function api(path, options) {
  await ensureToken();
  const opts = {
    ...options,
    headers: {
      ...(options?.headers || {}),
      Authorization: "Bearer " + state.token,
    },
  };
  let res = await fetch(path, opts);
  if (res.status === 401) {
    localStorage.removeItem("qg_token");
    state.token = "";
    const boot = await fetch("/api/bootstrap");
    if (boot.ok) {
      const data = await boot.json();
      state.token = data.token;
      syncAIStatus(data.ai);
      localStorage.setItem("qg_token", state.token);
      opts.headers.Authorization = "Bearer " + state.token;
      res = await fetch(path, opts);
    } else {
      await ensureToken();
      opts.headers.Authorization = "Bearer " + state.token;
      res = await fetch(path, opts);
    }
  }
  if (!res.ok) {
    let msg = await res.text();
    try {
      const body = JSON.parse(msg);
      if (body.error) msg = body.error;
    } catch { /* plain text */ }
    throw new Error(msg);
  }
  return res.json();
}

async function apiForm(path, formData) {
  await ensureToken();
  let res = await fetch(path, {
    method: "POST",
    headers: { Authorization: "Bearer " + state.token },
    body: formData,
  });
  if (res.status === 401) {
    localStorage.removeItem("qg_token");
    state.token = "";
    const boot = await fetch("/api/bootstrap");
    if (boot.ok) {
      const data = await boot.json();
      state.token = data.token;
      syncAIStatus(data.ai);
      localStorage.setItem("qg_token", state.token);
      res = await fetch(path, {
        method: "POST",
        headers: { Authorization: "Bearer " + state.token },
        body: formData,
      });
    } else {
      await ensureToken();
      res = await fetch(path, {
        method: "POST",
        headers: { Authorization: "Bearer " + state.token },
        body: formData,
      });
    }
  }
  if (!res.ok) {
    let msg = await res.text();
    try {
      const body = JSON.parse(msg);
      if (body.error) msg = body.error;
    } catch { /* plain text */ }
    throw new Error(msg);
  }
  return res.json();
}

function editorIssues(ctx) {
  if (ctx.archive && ctx.activeFile) {
    return ctx.issues.filter(i => i.file === ctx.activeFile);
  }
  return ctx.issues;
}

function setActiveArchiveFile(ctx, path) {
  ctx.activeFile = path;
  const file = ctx.files?.find(f => f.filename === path);
  if (file) {
    ctx.source = normalizeSource(file.text);
    ctx.filename = file.filename;
  }
  const visible = editorIssues(ctx);
  ctx.selectedLine = visible[0]?.line || null;
  syncEditorView();
  if (state.editorMode === "project") {
    renderStoredIssueList();
    renderStoredFixPanel();
  } else {
    renderPlaygroundFixPanel();
  }
}

function renderArchiveFilePicker(ctx, selectId) {
  if (!ctx.archive || !ctx.files?.length) return "";
  const opts = ctx.files.map(f => `
    <option value="${escapeHtml(f.filename)}" ${f.filename === ctx.activeFile ? "selected" : ""}>
      ${escapeHtml(f.filename)} (${ctx.issues.filter(i => i.file === f.filename).length} uyarı)
    </option>`).join("");
  return `
    <label class="archive-file-picker">
      <span>Dosya</span>
      <select id="${selectId}">${opts}</select>
    </label>`;
}

function bindArchiveFilePicker(ctx, selectId) {
  const sel = document.getElementById(selectId);
  if (!sel) return;
  sel.onchange = () => setActiveArchiveFile(ctx, sel.value);
}

function route() {
  const hash = location.hash.replace("#", "") || "/";
  if (hash === "/" || hash === "/overview") {
    renderOverview();
    return;
  }
  if (hash === "/playground") {
    renderPlayground();
    return;
  }
  if (hash === "/projects") {
    renderProjects();
    return;
  }
  if (hash === "/upload") {
    renderUpload();
    return;
  }
  if (hash === "/history") {
    renderHistory();
    return;
  }
  if (hash === "/chat") {
    if (!document.getElementById("chat-stream-bubble")) state.chatBusy = false;
    renderChat();
    return;
  }
  if (hash.startsWith("/project/")) {
    renderProject(decodeURIComponent(hash.split("/")[2] || ""));
    return;
  }
  renderOverview();
}

function formatWhen(iso) {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleString("tr-TR");
  } catch {
    return iso;
  }
}

function setPage(title, subtitle, actions) {
  document.getElementById("page-title").textContent = title;
  const sub = document.getElementById("page-subtitle");
  sub.textContent = subtitle || "";
  sub.classList.toggle("hidden", !subtitle);
  document.getElementById("topbar-actions").innerHTML = actions || `
    <button class="btn secondary" id="refresh-btn">Yenile</button>
  `;
  const refresh = document.getElementById("refresh-btn");
  if (refresh) refresh.onclick = () => route();
}

function renderNav(active) {
  const sections = [
    {
      items: [{ id: "overview", href: "#/", label: "Genel Bakış", icon: "◫" }],
    },
    {
      title: "Analiz",
      items: [
        { id: "playground", href: "#/playground", label: "Canlı Analiz", icon: "⌘" },
        { id: "upload", href: "#/upload", label: "Dosya Yükle", icon: "↑" },
      ],
    },
    {
      title: "Projeler",
      items: [
        { id: "projects", href: "#/projects", label: "Projeler", icon: "▦" },
        { id: "history", href: "#/history", label: "Geçmiş", icon: "◷" },
      ],
    },
    {
      title: "YZ",
      items: [{ id: "chat", href: "#/chat", label: "Sohbet", icon: "✦" }],
    },
  ];
  document.getElementById("nav").innerHTML = sections.map(section => `
    ${section.title ? `<div class="nav-section">${section.title}</div>` : ""}
    ${section.items.map(item => `
      <a class="nav-item ${active === item.id ? "active" : ""}" href="${item.href}">
        <span class="nav-icon">${item.icon}</span>${item.label}
      </a>`).join("")}
  `).join("");
}

function countBySeverity(issues) {
  const counts = { BLOCKER: 0, CRITICAL: 0, MAJOR: 0, MINOR: 0, INFO: 0 };
  for (const issue of issues) counts[issue.severity] = (counts[issue.severity] || 0) + 1;
  return counts;
}

function renderSeverityStats(issues) {
  const c = countBySeverity(issues);
  const rows = [
    { key: "total", label: "Toplam uyarı", value: issues.length, cls: "total" },
    { key: "BLOCKER", label: trSeverity("BLOCKER"), value: c.BLOCKER || 0 },
    { key: "CRITICAL", label: trSeverity("CRITICAL"), value: c.CRITICAL || 0 },
    { key: "MAJOR", label: trSeverity("MAJOR"), value: c.MAJOR || 0 },
    { key: "MINOR", label: trSeverity("MINOR"), value: c.MINOR || 0 },
  ];
  return `<div class="severity-stats">${rows.map(r => `
    <div class="sev-stat ${r.cls || ""}">
      <div class="n">${r.value}</div>
      <div class="l">${r.label}</div>
    </div>`).join("")}</div>`;
}

function showModal({ title, description, fields, submitLabel, onSubmit }) {
  const root = document.getElementById("modal-root");
  root.classList.remove("hidden");
  root.setAttribute("aria-hidden", "false");
  root.innerHTML = `
    <div class="modal" role="dialog">
      <h3>${escapeHtml(title)}</h3>
      <p>${escapeHtml(description)}</p>
      ${fields.map(f => `
        <label class="field">
          <span>${escapeHtml(f.label)}</span>
          <input id="modal-${f.id}" type="text" value="${escapeHtml(f.value || "")}" placeholder="${escapeHtml(f.placeholder || "")}" />
        </label>`).join("")}
      <div class="modal-actions">
        <button class="btn secondary" id="modal-cancel">İptal</button>
        <button class="btn primary" id="modal-submit">${escapeHtml(submitLabel || "Kaydet")}</button>
      </div>
    </div>`;

  const close = () => {
    root.classList.add("hidden");
    root.setAttribute("aria-hidden", "true");
    root.innerHTML = "";
  };

  root.onclick = e => { if (e.target === root) close(); };
  document.getElementById("modal-cancel").onclick = close;
  document.getElementById("modal-submit").onclick = async () => {
    const values = {};
    for (const f of fields) {
      values[f.id] = document.getElementById(`modal-${f.id}`).value.trim();
    }
    await onSubmit(values, close);
  };
}

async function renderOverview() {
  setPage("Genel Bakış");
  renderNav("overview");

  const root = document.getElementById("app-root");
  root.innerHTML = `<div class="empty"><span class="loading"></span></div>`;

  try {
    const [projects, history] = await Promise.all([
      api("/api/v1/projects/overview"),
      api("/api/v1/history"),
    ]);
    state.projects = projects;

    const openIssues = projects.reduce((n, p) => n + (p.open_issues || 0), 0);
    const vulns = projects.reduce((n, p) => n + (p.vulnerabilities || 0), 0);
    const failedGates = projects.filter(p => p.gate?.status === "FAIL").length;
    const recent = history.slice(0, 5);

    root.innerHTML = `
      <div class="hero-banner">
        <div class="hero-copy">
          <p class="hero-eyebrow">Statik analiz platformu</p>
          <h3>Kod kalitenizi yönetin</h3>
        </div>
        <div class="hero-actions">
          <a href="#/upload" class="btn primary">Proje yükle</a>
          <a href="#/playground" class="btn secondary">Canlı analiz</a>
        </div>
      </div>

      <div class="dashboard-grid">
        <div class="stat-card accent">
          <div class="stat-icon">◫</div>
          <div class="label">Proje</div>
          <div class="value">${projects.length}</div>
        </div>
        <div class="stat-card danger">
          <div class="stat-icon">!</div>
          <div class="label">Açık sorun</div>
          <div class="value">${openIssues}</div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">⬡</div>
          <div class="label">Güvenlik</div>
          <div class="value">${vulns}</div>
        </div>
        <div class="stat-card ${failedGates ? "danger" : "ok"}">
          <div class="stat-icon">${failedGates ? "✕" : "✓"}</div>
          <div class="label">Kapı</div>
          <div class="value">${failedGates || "OK"}</div>
        </div>
      </div>

      <div class="panel-section panel-compact">
        <h3>Örnek taramalar</h3>
        <div class="demo-grid">
          <button type="button" class="demo-card" data-demo="python-kritik">
            <div class="demo-card-top">
              <strong>Python · kritik</strong>
              <span class="gate-pill fail">KALIR</span>
            </div>
          </button>
          <button type="button" class="demo-card" data-demo="python-stil">
            <div class="demo-card-top">
              <strong>Python · stil</strong>
              <span class="gate-pill pass">GEÇER</span>
            </div>
          </button>
          <button type="button" class="demo-card" data-demo="javascript-stil">
            <div class="demo-card-top">
              <strong>JavaScript · stil</strong>
              <span class="gate-pill pass">GEÇER</span>
            </div>
          </button>
        </div>
      </div>

      ${projects.length ? `
      <div class="panel-section">
        <div class="panel-head">
          <h3>Projeler</h3>
          <a class="link-btn" href="#/projects">Tümü →</a>
        </div>
        <div class="project-quick-list">
          ${[...projects].sort((a, b) => (b.open_issues || 0) - (a.open_issues || 0)).slice(0, 5).map(p => `
            <a class="project-quick-row" href="#/project/${encodeURIComponent(p.key)}">
              <span class="project-quick-name">${escapeHtml(p.name)}</span>
              <span class="project-quick-meta">${p.open_issues} açık</span>
              ${p.gate ? `<span class="gate-pill ${p.gate.status.toLowerCase()}">${escapeHtml(p.gate.status_tr || trGate(p.gate.status))}</span>` : ""}
            </a>`).join("")}
        </div>
      </div>` : ""}

      <div class="panel-section">
        <div class="panel-head">
          <h3>Son taramalar</h3>
          ${recent.length ? `<a class="link-btn" href="#/history">Geçmiş →</a>` : ""}
        </div>
        ${recent.length ? `
          <div class="history-table-wrap">
            <table class="history-table">
              <thead>
                <tr><th>Tarih</th><th>Proje</th><th>Sorun</th><th>Kapı</th><th></th></tr>
              </thead>
              <tbody>
                ${recent.map(h => `
                  <tr>
                    <td>${escapeHtml(formatWhen(h.created_at))}</td>
                    <td><strong>${escapeHtml(h.project_name || h.project_key)}</strong></td>
                    <td>${h.issues_found}</td>
                    <td><span class="gate-pill ${(h.gate_status || "PASS").toLowerCase()}">${escapeHtml(h.gate_status_tr || trGate(h.gate_status))}</span></td>
                    <td><a class="link-btn" href="#/project/${encodeURIComponent(h.project_key)}">Detay</a></td>
                  </tr>`).join("")}
              </tbody>
            </table>
          </div>
        ` : `
          <div class="empty empty-compact">
            <p>Henüz tarama yok.</p>
            <a href="#/upload" class="btn primary">Dosya yükle</a>
          </div>`}
      </div>`;
    document.querySelectorAll("[data-demo]").forEach(btn => {
      btn.onclick = () => loadDemoToPlayground(btn.dataset.demo);
    });
  } catch (err) {
    root.innerHTML = `<div class="empty"><h3>Veri yüklenemedi</h3><p>${escapeHtml(err.message)}</p></div>`;
  }
}

function issuesByLine(issues) {
  const map = new Map();
  for (const issue of issues) {
    if (!map.has(issue.line)) map.set(issue.line, []);
    map.get(issue.line).push(issue);
  }
  return map;
}

function filteredIssues(issues) {
  return issues.filter(issue => {
    if (state.statusFilter === "OPEN" && issue.status === "CLOSED") return false;
    if (state.statusFilter === "SUPPRESSED") {
      if (issue.status !== "CLOSED") return false;
      if (issue.resolution !== "FALSE_POSITIVE" && issue.resolution !== "WONTFIX") return false;
    }
    if (state.filter !== "ALL" && issue.severity !== state.filter) return false;
    if (state.typeFilter !== "ALL" && issue.type !== state.typeFilter) return false;
    return true;
  });
}

async function setIssueResolution(issueId, resolution) {
  if (!state.projectKey) return;
  const result = await api(`/api/v1/projects/${encodeURIComponent(state.projectKey)}/issues/${encodeURIComponent(issueId)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ resolution }),
  });
  const idx = state.issues.findIndex(i => i.id === issueId);
  if (idx >= 0) state.issues[idx] = result.issue;
  state.projectGate = result.gate;
  const gateEl = document.querySelector(".project-gate");
  if (gateEl) {
    gateEl.innerHTML = renderGateBadge(result.gate, state.issues.filter(i => i.status !== "CLOSED"));
  }
  const visible = filteredIssues(state.issues);
  if (!visible.some(i => i.id === state.selectedId)) {
    state.selectedId = visible[0]?.id || null;
  }
  renderIssueFilters();
  renderStoredIssueList();
  renderStoredFixPanel();
}

function renderResolutionActions(issue) {
  if (!state.projectKey) return "";
  const suppressed = issue.status === "CLOSED" &&
    (issue.resolution === "FALSE_POSITIVE" || issue.resolution === "WONTFIX");
  if (suppressed) {
    return `
      <div class="resolution-box suppressed">
        <span class="resolution-badge">${escapeHtml(trResolution(issue.resolution))}</span>
        <p>Bu bulgu sonraki taramalarda yok sayılır.</p>
        <button class="btn secondary" id="reopen-issue">Geri al</button>
      </div>`;
  }
  if (issue.status === "CLOSED") return "";
  return `
    <div class="resolution-box">
      <div class="resolution-title">Bu bulgu hakkında</div>
      <div class="resolution-actions">
        <button class="btn ghost" id="mark-fp">Yanlış alarm</button>
        <button class="btn ghost" id="mark-wontfix">Düzeltilmeyecek</button>
      </div>
    </div>`;
}

function bindResolutionActions(issue) {
  document.getElementById("mark-fp")?.addEventListener("click", () =>
    setIssueResolution(issue.id, "FALSE_POSITIVE").catch(err => showToast(err.message, "error")));
  document.getElementById("mark-wontfix")?.addEventListener("click", () =>
    setIssueResolution(issue.id, "WONTFIX").catch(err => showToast(err.message, "error")));
  document.getElementById("reopen-issue")?.addEventListener("click", () =>
    setIssueResolution(issue.id, "REOPEN").catch(err => showToast(err.message, "error")));
}

function selectPlaygroundLine(lineNo) {
  selectEditorLine(lineNo);
}

function selectEditorLine(lineNo) {
  const ctx = editorCtx();
  ctx.selectedLine = lineNo;
  syncEditorView();
  scrollEditorToLine(lineNo);
  if (state.editorMode === "project") {
    renderStoredFixPanel();
  } else {
    renderPlaygroundFixPanel();
  }
}

function scrollEditorToLine(lineNo) {
  const textarea = document.getElementById("code-source");
  if (!textarea || !lineNo) return;
  const lineHeight = 22;
  const padding = 12;
  const target = Math.max(0, (lineNo - 3) * lineHeight - padding);
  textarea.scrollTop = target;
  syncEditorScroll();
}

function mountCodeEditor({ readonly = false } = {}) {
  const textarea = document.getElementById("code-source");
  if (!textarea) return;

  const ctx = editorCtx();
  ctx.source = normalizeSource(ctx.source || "");
  textarea.value = ctx.source;
  textarea.readOnly = readonly;
  textarea.classList.toggle("readonly", readonly);

  textarea.oninput = () => {
    if (readonly) return;
    ctx.source = textarea.value;
    if (state.editorMode === "playground") {
      localStorage.setItem("qg_playground_code", textarea.value);
      onPlaygroundSourceChanged();
    }
    syncEditorView();
  };
  textarea.onscroll = () => syncEditorScroll();
  textarea.onclick = () => {
    const lineNo = lineNumberAt(textarea, textarea.selectionStart);
    if (lineNo && issuesByLine(ctx.issues).has(lineNo)) {
      selectEditorLine(lineNo);
    }
  };

  syncEditorView();
}

function onPlaygroundSourceChanged() {
  const source = normalizeSource(state.playground.source);
  if (source === state.playground.lastAnalyzedSource) return;

  state.playground.issues = [];
  state.playground.gate = null;
  state.playground.selectedLine = null;
  state.playground.lastReport = null;

  const panel = document.getElementById("fix-panel");
  if (panel) {
    panel.innerHTML = `<div class="empty"><p>Kod güncellendi · analiz ediliyor...</p></div>`;
  }
  const saveBtn = document.getElementById("save-history-btn");
  if (saveBtn) saveBtn.disabled = true;
  setPlaygroundSummary("Kod değişti · analiz ediliyor...");
  schedulePlaygroundAnalysis();
}

function setPlaygroundSummary(text) {
  const el = document.getElementById("playground-summary");
  if (el) el.textContent = text;
}

function schedulePlaygroundAnalysis() {
  clearTimeout(state.playground.analyzeTimer);
  state.playground.analyzeTimer = setTimeout(() => {
    runPlayground({ auto: true });
  }, 700);
}

function renderPlayground() {
  state.editorMode = "playground";
  setPage("Canlı Analiz", "Otomatik analiz", `
    <button class="btn secondary" id="save-history-btn" ${state.playground.lastReport ? "" : "disabled"}>Geçmişe Kaydet</button>
    <button class="btn primary" id="run-btn">Şimdi Analiz Et</button>
  `);
  renderNav("playground");

  const root = document.getElementById("app-root");
  root.innerHTML = `
    <div class="workspace playground">
      <div class="editor-panel">
        <div class="panel-title">Kaynak kod</div>
        <div class="editor-shell">
          <div class="gutter" id="gutter"></div>
          <div class="code-editor">
            <div class="code-highlights" id="code-highlights" aria-hidden="true"></div>
            <textarea id="code-source" spellcheck="false"></textarea>
          </div>
        </div>
        <div class="editor-hint" id="playground-summary"></div>
      </div>
      <div class="fix-panel" id="fix-panel">
        <div class="empty"><p>Kod yükleniyor...</p></div>
      </div>
    </div>`;

  mountCodeEditor({ readonly: false });

  document.getElementById("run-btn").onclick = () => runPlayground({ auto: false });
  const saveBtn = document.getElementById("save-history-btn");
  if (saveBtn) saveBtn.onclick = savePlaygroundToHistory;
  schedulePlaygroundAnalysis();
}

function updatePlaygroundSummary() {
  const issues = state.playground.issues;
  if (!issues.length) {
    const lang = state.playground.language ? ` · ${state.playground.language}` : "";
    setPlaygroundSummary(`Sorun bulunamadı — kod temiz görünüyor${lang}`);
    return;
  }

  const lineCount = new Set(issues.map(i => i.line)).size;
  const hasSyntax = issues.some(i => i.rule_key === "python:syntax-error");
  const b = issueBreakdown(issues);
  const lang = state.playground.language ? ` · ${state.playground.language}` : "";
  let text = `${issues.length} uyarı · ${lineCount} satır · ${b.smells} stil${lang}`;
  if (b.blocker + b.critical + b.vuln > 0) {
    text = `${issues.length} sorun · ${b.blocker + b.critical} kritik · ${lineCount} satır${lang}`;
  }
  if (hasSyntax && issues.length > 1) {
    text += " · kısmi tarama";
  }
  setPlaygroundSummary(text);
}

function issueBreakdown(issues) {
  const b = { total: issues.length, blocker: 0, critical: 0, bugs: 0, vuln: 0, smells: 0 };
  for (const i of issues) {
    if (i.severity === "BLOCKER") b.blocker++;
    if (i.severity === "CRITICAL") b.critical++;
    if (i.type === "BUG") b.bugs++;
    if (i.type === "VULNERABILITY") b.vuln++;
    if (i.type === "CODE_SMELL") b.smells++;
  }
  return b;
}

function renderGateBadge(gate, issues) {
  if (!gate) return "";
  const b = issueBreakdown(issues || []);
  const cls = gate.status === "PASS" ? "pass" : gate.status === "WARN" ? "warn" : "fail";
  return `
    <div class="gate-card ${cls}">
      <div class="gate-title">${escapeHtml(gate.name_tr || gate.name)}</div>
      ${b.total > 0 ? `
      <div class="gate-totals">
        <div class="gate-total-number">${b.total}</div>
        <div class="gate-total-label">uyarı</div>
        <div class="gate-total-split">kritik ${b.blocker + b.critical + b.vuln} · stil ${b.smells || b.total}</div>
      </div>` : ""}
      <div class="gate-status">${escapeHtml(gate.status_tr || trGate(gate.status))}</div>
      <div class="gate-section-label">Kriterler</div>
      <div class="gate-conditions">
        ${(gate.conditions || []).map(c => `
          <div class="gate-row ${c.passed ? "ok" : "bad"}">
            <span>${c.passed ? "✓" : "✗"}</span>
            <span>${escapeHtml(c.label_tr)}: <strong>${c.actual}</strong> / ${c.threshold}</span>
          </div>
        `).join("")}
      </div>
    </div>`;
}

function renderAnalysisSummary(issues, gate) {
  return `<div class="analysis-summary">${renderGateBadge(gate, issues)}</div>`;
}

function issueAIKey(issue, suffix = "") {
  const base = issue.id || issue.fingerprint || `${issue.file}-${issue.line}-${issue.rule_key}`;
  return String(base + suffix).replace(/[^a-zA-Z0-9_-]/g, "_");
}

function renderAIHelpUI(issueKey) {
  return `
    <div class="ai-help">
      <button type="button" class="btn ai-help-btn" data-ai-toggle="${issueKey}">
        <span class="ai-icon">✦</span> YZ'ye sor
      </button>
      <div class="ai-ask-panel hidden" id="ai-ask-${issueKey}">
        <input type="text" class="ai-ask-input" id="ai-input-${issueKey}"
          placeholder="Sorunuzu yazın…" />
        <div class="ai-ask-actions">
          <button type="button" class="btn primary ai-send-btn" data-ai-send="${issueKey}">Sor</button>
        </div>
        <div class="ai-answer hidden" id="ai-answer-${issueKey}"></div>
      </div>
    </div>`;
}

function formatRichText(text) {
  if (!text) return "";
  const blocks = [];
  const re = /```([\s\S]*?)```/g;
  let last = 0;
  let match;
  while ((match = re.exec(text)) !== null) {
    if (match.index > last) blocks.push({ kind: "text", value: text.slice(last, match.index) });
    let code = match[1].trim().replace(/^[a-zA-Z0-9_-]+\n/, "");
    blocks.push({ kind: "code", value: code });
    last = match.index + match[0].length;
  }
  if (last < text.length) blocks.push({ kind: "text", value: text.slice(last) });

  return blocks.map(b => {
    if (b.kind === "code") {
      return `<pre class="chat-code"><code>${escapeHtml(b.value)}</code></pre>`;
    }
    let t = escapeHtml(b.value.trim());
    t = t.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
    t = t.replace(/^- (.+)$/gm, "<li>$1</li>");
    t = t.replace(/(<li>[\s\S]*?<\/li>)+/g, m => `<ul class="chat-list">${m}</ul>`);
    t = t.replace(/`([^`]+)`/g, '<code class="chat-inline">$1</code>');
    t = t.replace(/\n\n/g, "</p><p>");
    t = t.replace(/\n/g, "<br>");
    return t ? `<p>${t}</p>` : "";
  }).join("");
}

function formatAIAnswerHTML(result) {
  const exp = result.explanation || result;
  const mode = result.llm_active
    ? (result.provider ? `${result.provider}${result.model ? " · " + result.model : ""}` : aiProviderLabel() || "AI")
    : "Hızlı yanıt";
  let body = "";
  if (exp.summary_tr) body += `<div class="ai-block">${formatRichText(exp.summary_tr)}</div>`;
  if (exp.risk_tr) body += `<div class="ai-block ai-risk">${formatRichText(exp.risk_tr)}</div>`;
  if (exp.example_tr) body += `<div class="ai-block ai-example">${formatRichText(exp.example_tr)}</div>`;
  if (!body) body = `<p>Yanıt alınamadı.</p>`;
  return `<div class="ai-answer-body">${body}<span class="ai-meta">${mode}</span></div>`;
}

async function askAI(issueKey, issue, codeLine, question) {
  const box = document.getElementById(`ai-answer-${issueKey}`);
  const btn = document.querySelector(`[data-ai-send="${issueKey}"]`);
  if (!box) return;
  const q = (question || "").trim();
  if (!q) {
    box.classList.remove("hidden");
    box.innerHTML = `<p class="ai-hint-text">Önce kendi sorunu yaz.</p>`;
    return;
  }
  box.classList.remove("hidden");
  const cacheKey = `${issueKey}:${q}`;
  if (state.aiCache[cacheKey]) {
    box.innerHTML = formatAIAnswerHTML(state.aiCache[cacheKey]);
    return;
  }
  box.innerHTML = `<p class="ai-loading"><span class="loading"></span> Yanıtlanıyor...</p>`;
  if (btn) btn.disabled = true;
  try {
    const result = await api("/api/v1/explain/issue", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        issue,
        code_line: codeLine || "",
        question: q,
      }),
    });
    state.aiCache[cacheKey] = result;
    box.innerHTML = formatAIAnswerHTML(result);
  } catch (err) {
    box.innerHTML = `<p class="ai-error">${escapeHtml(err.message)}</p>`;
  } finally {
    if (btn) btn.disabled = false;
  }
}

function bindAIHelp(issue, codeLine, suffix = "") {
  const key = issueAIKey(issue, suffix);
  document.querySelector(`[data-ai-toggle="${key}"]`)?.addEventListener("click", () => {
    const panel = document.getElementById(`ai-ask-${key}`);
    if (!panel) return;
    panel.classList.toggle("hidden");
    if (!panel.classList.contains("hidden")) {
      document.getElementById(`ai-input-${key}`)?.focus();
    }
  });
  const send = () => {
    const input = document.getElementById(`ai-input-${key}`);
    askAI(key, issue, codeLine, input?.value?.trim() || "");
  };
  document.querySelector(`[data-ai-send="${key}"]`)?.addEventListener("click", send);
  document.getElementById(`ai-input-${key}`)?.addEventListener("keydown", e => {
    if (e.key === "Enter") {
      e.preventDefault();
      send();
    }
  });
}

const CHAT_SUGGESTIONS = [
  "SQL injection nedir?",
  "Kalite kapısı ne zaman kırılır?",
  "eval kullanımı neden tehlikeli?",
  "QualiGuard ne işe yarar?",
];

function renderChat() {
  renderNav("chat");
  setPage("Sohbet", state.aiActive ? aiProviderLabel() : "YZ kapalı");
  const status = state.aiActive
    ? `<span class="chat-status ok">Aktif</span>`
    : `<span class="chat-status off">Kapalı</span>`;

  const intro = state.chatMessages.length === 0
    ? `<div class="chat-bubble assistant">
        <p>Güvenlik, kod kalitesi ve kalite kapısı hakkında sorabilirsiniz.</p>
      </div>
      <div class="chat-suggestions">${CHAT_SUGGESTIONS.map(s => `
        <button type="button" class="chat-suggestion" data-suggest="${escapeHtml(s)}">${escapeHtml(s)}</button>`).join("")}</div>`
    : "";

  const history = state.chatMessages.map(m => `
    <div class="chat-bubble ${m.role === "user" ? "user" : "assistant"}">
      ${m.role === "assistant" ? formatRichText(m.content) : `<p>${escapeHtml(m.content).replaceAll("\n", "<br>")}</p>`}
    </div>`).join("");

  const canRetry = state.chatMessages.length >= 2 &&
    state.chatMessages[state.chatMessages.length - 1].role === "assistant" &&
    !state.chatBusy;

  document.getElementById("app-root").innerHTML = `
    <section class="chat-page">
      <div class="chat-toolbar">${status}
        ${canRetry ? `<button type="button" class="btn ghost" id="chat-retry">Yeniden dene</button>` : ""}
        <button type="button" class="btn secondary" id="chat-clear">Temizle</button>
      </div>
      <div class="chat-thread" id="chat-thread">${intro}${history}</div>
      <form class="chat-compose" id="chat-form">
        <textarea id="chat-input" rows="3" placeholder="Mesaj…" ${state.aiActive ? "" : "disabled"}></textarea>
        <button type="submit" class="btn primary" id="chat-send" ${state.aiActive ? "" : "disabled"}>Gönder</button>
      </form>
    </section>`;

  document.getElementById("chat-retry")?.addEventListener("click", retryLastChat);

  document.getElementById("chat-clear")?.addEventListener("click", () => {
    state.chatMessages = [];
    sessionStorage.removeItem("qg_chat");
    renderChat();
  });

  document.querySelectorAll("[data-suggest]").forEach(btn => {
    btn.onclick = () => sendChatMessage(btn.dataset.suggest);
  });

  const form = document.getElementById("chat-form");
  const input = document.getElementById("chat-input");
  form?.addEventListener("submit", e => {
    e.preventDefault();
    sendChatMessage(input?.value || "");
  });
  input?.addEventListener("keydown", e => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendChatMessage(input.value || "");
    }
  });
  scrollChatToBottom();
}

function scrollChatToBottom() {
  const el = document.getElementById("chat-thread");
  if (el) el.scrollTop = el.scrollHeight;
}

function friendlyChatError(err) {
  const msg = String(err?.message || err || "");
  if (/model.*not found|not_found_error/i.test(msg)) {
    return "Ollama'da seçili model yüklü değil. Terminalde: ollama list ile bakın, eksikse ollama pull llama3.2 — ardından sunucuyu yeniden başlatın.";
  }
  if (/connection refused|ECONNREFUSED|fetch failed/i.test(msg)) {
    return "Ollama çalışmıyor gibi görünüyor. Ollama uygulamasını açıp server.bat'ı yeniden başlatın.";
  }
  return `Üzgünüm, yanıt veremedim: ${msg}`;
}

async function retryLastChat() {
  if (state.chatBusy || state.chatMessages.length < 2) return;
  const last = state.chatMessages[state.chatMessages.length - 1];
  if (last.role !== "assistant") return;
  state.chatMessages.pop();
  sessionStorage.setItem("qg_chat", JSON.stringify(state.chatMessages));
  state.chatBusy = true;
  try {
    renderChat();
    const thread = document.getElementById("chat-thread");
    if (thread) {
      thread.insertAdjacentHTML("beforeend", `<div class="chat-bubble assistant" id="chat-stream-bubble"><p><span class="loading"></span></p></div>`);
      scrollChatToBottom();
    }
    const full = await fetchChatReply(state.chatMessages);
    state.chatMessages.push({ role: "assistant", content: full });
    sessionStorage.setItem("qg_chat", JSON.stringify(state.chatMessages));
    updateStreamBubble(full, false);
    document.getElementById("chat-stream-bubble")?.removeAttribute("id");
  } catch (err) {
    const errMsg = friendlyChatError(err);
    state.chatMessages.push({ role: "assistant", content: errMsg });
    sessionStorage.setItem("qg_chat", JSON.stringify(state.chatMessages));
    const bubble = document.getElementById("chat-stream-bubble");
    if (bubble) bubble.innerHTML = `<p class="chat-error">${escapeHtml(errMsg)}</p>`;
    else renderChat();
  } finally {
    state.chatBusy = false;
    document.getElementById("chat-input")?.focus();
  }
}

function updateStreamBubble(text, streaming) {
  const el = document.getElementById("chat-stream-bubble");
  if (!el) return;
  if (streaming) {
    el.innerHTML = text
      ? `<p>${escapeHtml(text).replaceAll("\n", "<br>")}<span class="chat-cursor">▍</span></p>`
      : `<p><span class="loading"></span></p>`;
  } else {
    el.innerHTML = formatRichText(text) || `<p>…</p>`;
  }
  scrollChatToBottom();
}

async function sendChatMessage(text) {
  const msg = text.trim();
  if (!msg || state.chatBusy || !state.aiActive) return;
  state.chatMessages.push({ role: "user", content: msg });
  state.chatBusy = true;

  try {
    renderChat();
    const input = document.getElementById("chat-input");
    if (input) input.value = "";

    const thread = document.getElementById("chat-thread");
    if (thread) {
      thread.insertAdjacentHTML("beforeend", `<div class="chat-bubble assistant" id="chat-stream-bubble"><p><span class="loading"></span></p></div>`);
      scrollChatToBottom();
    }

    let full = await fetchChatReply(state.chatMessages);
    state.chatMessages.push({ role: "assistant", content: full });
    sessionStorage.setItem("qg_chat", JSON.stringify(state.chatMessages));
    updateStreamBubble(full, false);
    document.getElementById("chat-stream-bubble")?.removeAttribute("id");
  } catch (err) {
    const errMsg = friendlyChatError(err);
    state.chatMessages.push({ role: "assistant", content: errMsg });
    sessionStorage.setItem("qg_chat", JSON.stringify(state.chatMessages));
    const bubble = document.getElementById("chat-stream-bubble");
    if (bubble) bubble.innerHTML = `<p class="chat-error">${escapeHtml(errMsg)}</p>`;
    else renderChat();
  } finally {
    state.chatBusy = false;
    document.getElementById("chat-input")?.focus();
  }
}

async function fetchChatReply(messages) {
  try {
    return await fetchChatReplyStream(messages);
  } catch (streamErr) {
    console.warn("Stream başarısız, normal mod deneniyor:", streamErr);
    await ensureToken();
    const result = await api("/api/v1/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ messages }),
    });
    const reply = (result.reply || "").trim();
    if (!reply) throw new Error("Boş yanıt alındı");
    return reply;
  }
}

async function fetchChatReplyStream(messages) {
  await ensureToken();
  const res = await fetch("/api/v1/chat", {
    method: "POST",
    headers: {
      Authorization: "Bearer " + state.token,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ messages, stream: true }),
  });
  if (!res.ok) {
    const errText = await res.text();
    throw new Error(errText || `HTTP ${res.status}`);
  }
  if (!res.body) throw new Error("Stream desteklenmiyor");

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let full = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split("\n\n");
    buffer = events.pop() || "";
    for (const event of events) {
      for (const line of event.split("\n")) {
        const trimmed = line.trim();
        if (!trimmed.startsWith("data:")) continue;
        const payload = trimmed.slice(5).trim();
        if (!payload) continue;
        let json;
        try { json = JSON.parse(payload); } catch { continue; }
        if (json.error) throw new Error(json.error);
        if (json.delta) {
          full += json.delta;
          updateStreamBubble(full, true);
        }
        if (json.done) full = json.done;
      }
    }
  }

  // Kalan buffer
  if (buffer.trim()) {
    for (const line of buffer.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed.startsWith("data:")) continue;
      const payload = trimmed.slice(5).trim();
      if (!payload) continue;
      try {
        const json = JSON.parse(payload);
        if (json.error) throw new Error(json.error);
        if (json.delta) full += json.delta;
        if (json.done) full = json.done;
      } catch (e) {
        if (e.message && !e.message.includes("JSON")) throw e;
      }
    }
  }

  full = (full || "").trim();
  if (!full) throw new Error("Boş yanıt alındı");
  return full;
}

function renderGateHistory(history) {
  if (!history || !history.length) return "";
  return `
    <div class="gate-history">
      <div class="gate-title">Bu projenin son taramaları</div>
      ${history.map(h => `
        <div class="history-row">
          <span class="gate-pill ${(h.gate_status || "PASS").toLowerCase()}">${escapeHtml(h.gate_status_tr || trGate(h.gate_status))}</span>
          <span class="meta">${escapeHtml(formatWhen(h.created_at))} · ${h.open_issues ?? h.issues_open ?? 0} açık sorun</span>
        </div>
      `).join("")}
      <a class="link-btn" href="#/history">Tüm geçmişi gör →</a>
    </div>`;
}

function normalizeSource(source) {
  return String(source || "").replace(/\r\n/g, "\n").replace(/\r/g, "\n");
}

function lineAt(source, lineNo) {
  const lines = normalizeSource(source).split("\n");
  return lines[lineNo - 1] ?? "";
}

function issueCodeLine(issue, source) {
  if (issue.snippet) {
    const marked = issue.snippet.split("\n").find(l => l.startsWith("> "));
    if (marked) {
      return marked.replace(/^>\s*\d+\s*\|\s?/, "");
    }
  }
  return lineAt(source, issue.line);
}

function lineNumberAt(textarea, index) {
  const value = textarea.value.slice(0, index);
  return value.split("\n").length;
}

function syncEditorScroll() {
  const textarea = document.getElementById("code-source");
  const gutter = document.getElementById("gutter");
  const highlights = document.getElementById("code-highlights");
  if (!textarea) return;
  if (gutter) gutter.scrollTop = textarea.scrollTop;
  if (highlights) highlights.scrollTop = textarea.scrollTop;
}

function syncEditorView() {
  const textarea = document.getElementById("code-source");
  const gutter = document.getElementById("gutter");
  const highlights = document.getElementById("code-highlights");
  if (!textarea || !gutter || !highlights) return;

  const ctx = editorCtx();
  const lines = normalizeSource(ctx.source).split("\n");
  const lineMap = issuesByLine(editorIssues(ctx));

  gutter.innerHTML = lines.map((_, i) => {
    const lineNo = i + 1;
    const hasError = lineMap.has(lineNo);
    const active = ctx.selectedLine === lineNo ? " active" : "";
    return `<div class="gutter-line ${hasError ? "error" : ""}${active}" data-line="${lineNo}">${lineNo}</div>`;
  }).join("");

  highlights.innerHTML = lines.map((line, i) => {
    const lineNo = i + 1;
    const hasError = lineMap.has(lineNo);
    const active = ctx.selectedLine === lineNo ? " active" : "";
    const text = line === "" ? " " : escapeHtml(line);
    return `<div class="code-line ${hasError ? "error" : ""}${active}" data-line="${lineNo}">${text}</div>`;
  }).join("");

  gutter.querySelectorAll(".gutter-line.error").forEach(el => {
    el.onclick = () => selectPlaygroundLine(Number(el.dataset.line));
  });
  highlights.querySelectorAll(".code-line.error").forEach(el => {
    el.onclick = () => selectPlaygroundLine(Number(el.dataset.line));
  });

  syncEditorScroll();
}

function syncGutter() {
  syncEditorView();
}

async function runPlayground({ auto = false } = {}) {
  const gen = ++state.playground.analyzeGen;
  const source = normalizeSource(state.playground.source);
  state.playground.analyzing = true;

  const panel = document.getElementById("fix-panel");
  const runBtn = document.getElementById("run-btn");
  if (runBtn) runBtn.disabled = true;
  if (!auto && panel) {
    panel.innerHTML = `<div class="empty"><p>Analiz ediliyor...</p></div>`;
  }
  setPlaygroundSummary("Analiz ediliyor...");

  try {
    const report = await api("/api/v1/analyze/code", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        language: "auto",
        source,
      }),
    });

    if (gen !== state.playground.analyzeGen) return;

    state.playground.issues = report.issues || [];
    state.playground.gate = report.gate || null;
    state.playground.lastReport = report;
    state.playground.language = report.source?.language || "";
    state.playground.lastAnalyzedSource = source;
    if (state.playground.issues.length > 0) {
      state.playground.selectedLine = state.playground.issues[0].line;
    } else {
      state.playground.selectedLine = null;
    }

    syncEditorView();
    updatePlaygroundSummary();
    renderPlaygroundFixPanel();
    const saveBtn = document.getElementById("save-history-btn");
    if (saveBtn) saveBtn.disabled = false;
  } catch (err) {
    if (gen !== state.playground.analyzeGen) return;
    const friendly = String(err.message).includes("Python")
      ? err.message
      : `Analiz başarısız: ${err.message}`;
    if (panel) {
      panel.innerHTML = `<div class="empty"><h3>Analiz yapılamadı</h3><p>${escapeHtml(friendly)}</p><p class="meta">Python 3 kurulu olmalı. Sunucuyu server.bat ile yeniden başlatın.</p></div>`;
    }
    setPlaygroundSummary("Analiz başarısız");
  } finally {
    if (gen === state.playground.analyzeGen) {
      state.playground.analyzing = false;
      if (runBtn) runBtn.disabled = false;
    }
  }
}

function renderPlaygroundFixPanel() {
  const panel = document.getElementById("fix-panel");
  const ctx = editorCtx();
  const visibleIssues = editorIssues(ctx);
  const line = ctx.selectedLine;
  const lineIssues = visibleIssues.filter(i => i.line === line);

  if (!visibleIssues.length) {
    panel.innerHTML = `
      <div class="empty success">
        <h3>Sorun bulunamadı</h3>
        <p>Bu kod parçasında QualiGuard kurallarına göre ihlal tespit edilmedi.</p>
        ${state.editorMode !== "upload" ? renderGateBadge(ctx.gate, ctx.issues) : ""}
      </div>`;
    return;
  }

  if (!line || !lineIssues.length) {
    panel.innerHTML = `
      ${state.editorMode !== "upload" ? renderGateBadge(ctx.gate, ctx.issues) : ""}
      <div class="empty"><p>Detay görmek için vurgulu bir satır seçin</p></div>`;
    return;
  }

  panel.innerHTML = `
    ${state.editorMode !== "upload" ? renderGateBadge(ctx.gate, ctx.issues) : ""}
    ${lineIssues.map((issue, idx) => {
      const codeLine = issueCodeLine(issue, ctx.source);
      return `
    <div class="issue-block">
      ${lineIssues.length > 1 ? `<p class="meta">Sorun ${idx + 1} / ${lineIssues.length} (satır ${issue.line})</p>` : ""}
      <div class="issue-head">
        <span class="badge ${issue.severity}">${trSeverity(issue.severity)}</span>
        <span class="type-tag">${trType(issue.type)}</span>
      </div>
      <h3>${escapeHtml(issue.message)}</h3>

      <div>
        <div class="code-head"><span>Mevcut kod (satır ${issue.line})</span></div>
        <pre class="code-block bad-line">${escapeHtml(codeLine || "(boş satır)")}</pre>
      </div>

      <div>
        <div class="code-head">
          <span>Önerilen düzeltme</span>
          <button class="btn primary copy-fix" data-idx="${idx}">Kopyala</button>
        </div>
        <pre class="code-block fix">${escapeHtml(issue.fix_suggestion || fixFallback(issue))}</pre>
      </div>
      ${renderAIHelpUI(issueAIKey(issue, `-${idx}`))}
    </div>
  `;
    }).join("")}`;

  panel.querySelectorAll(".copy-fix").forEach(btn => {
    btn.onclick = async () => {
      const issue = lineIssues[Number(btn.dataset.idx)];
      await navigator.clipboard.writeText(issue.fix_suggestion || fixFallback(issue));
      btn.textContent = "Kopyalandı!";
      setTimeout(() => { btn.textContent = "Kopyala"; }, 1500);
    };
  });
  lineIssues.forEach((issue, idx) => {
    const codeLine = issueCodeLine(issue, ctx.source);
    const key = issueAIKey(issue, `-${idx}`);
    bindAIHelp(issue, codeLine, `-${idx}`);
  });
}

function fixFallback(issue) {
  return `# Kural: ${issue.rule_key}
# Mesaj: ${issue.message}
#
# Ne yapın:
# 1. Satır ${issue.line} üzerindeki kodu inceleyin
# 2. Kural mesajındaki riski giderin
# 3. Tekrar analiz edin`;
}

function explainIssue(issue) {
  if (issue.rule_key?.startsWith("eslint:")) {
    const rule = issue.rule_key.slice(7);
    return `ESLint (${rule}): ${issue.message}`;
  }
  const map = {
    "python:sql-injection": "Kullanıcı girdisi SQL sorgusuna doğrudan ekleniyor. Parametreli sorgu kullanın.",
    "python:command-injection": "Değişken shell komutuna aktarılıyor. subprocess.run ile sabit argüman listesi tercih edin.",
    "python:bare-except": "Genel except tüm hataları gizler. Spesifik exception yakalayın.",
    "python:eval-usage": "eval/exec dinamik kod çalıştırır — kod enjeksiyonu riski taşır.",
    "python:hardcoded-password": "Gizli bilgi kaynak kodda sabit. Ortam değişkeni veya secret manager kullanın.",
    "python:unused-variable": "Tanımlanan değişken kullanılmıyor.",
    "python:unused-import": "Import edilen modül kullanılmıyor.",
    "python:empty-except": "Boş except bloğu hatayı yutar.",
    "python:pickle-usage": "pickle güvenilmeyen veriyle kullanılmamalı.",
    "python:weak-hash": "MD5/SHA1 kriptografik amaçla yeterli değil.",
    "python:debug-breakpoint": "breakpoint/debug kodu production'da kalmamalı.",
    "python:assert-usage": "assert optimizasyon modunda devre dışı kalabilir.",
    "javascript:innerhtml-xss": "Dinamik HTML atanıyor. textContent veya sanitize kullanın.",
    "javascript:innerhtml-static": "Statik innerHTML çalışır; createElement daha güvenli ve okunaklı.",
    "javascript:no-var": "var yerine let veya const kullanın.",
    "javascript:eqeqeq": "== yerine === kullanın (tip güvenliği).",
    "javascript:no-console": "console.log production kodda kalmamalı.",
  };
  return map[issue.rule_key] || issue.message;
}

async function renderDashboard() {
  location.hash = "#/projects";
}

async function uploadReport(report) {
  return api("/api/v1/analyses", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(report),
  });
}

async function savePlaygroundToHistory() {
  if (!state.playground.lastReport) return;

  showModal({
    title: "Geçmişe kaydet",
    description: "Bu analiz sonucunu proje olarak kaydedin.",
    fields: [
      { id: "key", label: "Proje kodu", value: "canli-analiz", placeholder: "ornek-proje" },
      { id: "name", label: "Proje adı", value: "Canlı Analiz", placeholder: "Proje adı" },
    ],
    submitLabel: "Kaydet",
    onSubmit: async (values, close) => {
      if (!values.key || !values.name) return;
      const report = structuredClone(state.playground.lastReport);
      report.project = { key: values.key, name: values.name };
      try {
        const result = await uploadReport(report);
        close();
        state.projects = await api("/api/v1/projects/overview");
        location.hash = "#/history";
      } catch (err) {
        showToast("Kayıt hatası: " + err.message, "error");
      }
    },
  });
}

async function renderUpload() {
  setPage("Dosya Yükle", "Tek dosya veya zip");
  renderNav("upload");
  state.uploadPreview.ready = false;

  const root = document.getElementById("app-root");
  root.innerHTML = `
    <div class="upload-panel">
      <div class="drop-zone" id="drop-zone">
        <p class="drop-title">Sürükle bırak veya dosya seç</p>
        <p class="meta">.py · .js · .ts · .go · .java · .cs · .zip</p>
        <label class="file-btn">
          Dosya Seç
          <input id="upload-file" type="file" accept=".py,.txt,.js,.jsx,.ts,.tsx,.go,.java,.cs,.zip,text/plain,application/zip" hidden />
        </label>
        <p class="meta" id="file-name">Seçili dosya yok</p>
      </div>

      <label class="field">
        <span>Proje kodu</span>
        <input id="upload-key" type="text" placeholder="Boş bırak = dosya adından" />
      </label>
      <label class="field">
        <span>Proje adı</span>
        <input id="upload-name" type="text" placeholder="Boş bırak = dosya adı" />
      </label>
      <button class="btn primary" id="upload-submit">Analiz Et</button>
      <p id="upload-status" class="upload-status"></p>
    </div>
    <div id="upload-preview-wrap" hidden></div>`;

  const fileInput = document.getElementById("upload-file");
  const fileNameEl = document.getElementById("file-name");
  const keyInput = document.getElementById("upload-key");
  const nameInput = document.getElementById("upload-name");
  const dropZone = document.getElementById("drop-zone");
  let pickedFile = null;
  let lastSource = "";

  function guessKey(filename) {
    return filename.replace(/\.[^.]+$/, "").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "kaynak-dosyasi";
  }

  function setPickedFile(file) {
    if (!file) return;
    pickedFile = file;
    fileNameEl.textContent = "Seçildi: " + file.name;
    dropZone.classList.add("has-file");
    if (!keyInput.value.trim()) keyInput.placeholder = guessKey(file.name);
    if (!nameInput.value.trim()) nameInput.placeholder = file.name.replace(/\.[^.]+$/, "");
  }

  fileInput.onchange = () => {
    if (fileInput.files?.[0]) setPickedFile(fileInput.files[0]);
  };

  dropZone.ondragover = e => { e.preventDefault(); dropZone.classList.add("drag"); };
  dropZone.ondragleave = () => dropZone.classList.remove("drag");
  dropZone.ondrop = e => {
    e.preventDefault();
    dropZone.classList.remove("drag");
    if (e.dataTransfer.files?.[0]) setPickedFile(e.dataTransfer.files[0]);
  };

  function isZipFile(file) {
    return file && /\.zip$/i.test(file.name);
  }

  function showUploadPreview(preview) {
    const isArchive = !!preview.archive && preview.files?.length;
    const activeFile = isArchive ? preview.files[0].filename : (preview.filename || "");
    const activeSource = isArchive ? preview.files[0].text : preview.source;

    state.editorMode = "upload";
    state.uploadPreview = {
      source: normalizeSource(activeSource),
      issues: preview.issues || [],
      selectedLine: (preview.issues && preview.issues[0]?.line) || null,
      gate: preview.gate || null,
      filename: isArchive ? activeFile : (preview.filename || ""),
      language: preview.language || "",
      ready: true,
      archive: isArchive,
      files: preview.files || [],
      activeFile,
    };
    if (isArchive) {
      const firstIssue = (preview.issues || []).find(i => i.file === activeFile);
      state.uploadPreview.selectedLine = firstIssue?.line || null;
    }

    const wrap = document.getElementById("upload-preview-wrap");
    wrap.hidden = false;
    const summaryMeta = isArchive
      ? `${preview.file_count || preview.files.length} dosya · ${(preview.issues || []).length} uyarı`
      : `${escapeHtml(preview.filename)} · ${escapeHtml(preview.language || "python")}`;
    wrap.innerHTML = `
      <div class="preview-summary">
        ${renderAnalysisSummary(preview.issues || [], preview.gate)}
        <p class="meta">${summaryMeta}</p>
        ${renderArchiveFilePicker(state.uploadPreview, "upload-file-select")}
      </div>
      <div class="workspace playground upload-preview">
        <div class="editor-panel">
          <div class="panel-title">Yüklenen kod — sorunlu satırlara tıklayın</div>
          <div class="editor-shell code-overview">
            <div class="gutter" id="gutter"></div>
            <div class="code-editor">
              <div class="code-highlights" id="code-highlights" aria-hidden="true"></div>
              <textarea id="code-source" spellcheck="false" readonly></textarea>
            </div>
          </div>
        </div>
        <div class="fix-panel" id="fix-panel"></div>
      </div>
      <div class="upload-actions">
        <button class="btn primary" id="upload-save">Projeye Kaydet</button>
        <button class="btn ghost" id="upload-back">← Yeni dosya</button>
      </div>`;

    mountCodeEditor({ readonly: true });
    bindArchiveFilePicker(state.uploadPreview, "upload-file-select");
    renderPlaygroundFixPanel();

    document.getElementById("upload-back").onclick = () => {
      wrap.hidden = true;
      wrap.innerHTML = "";
      state.uploadPreview.ready = false;
    };

    document.getElementById("upload-save").onclick = async () => {
      const status = document.getElementById("upload-status");
      status.textContent = "Kaydediliyor...";
      status.className = "upload-status";
      try {
        let result;
        if (isZipFile(pickedFile)) {
          const fd = new FormData();
          fd.append("file", pickedFile);
          const key = keyInput.value.trim();
          const name = nameInput.value.trim();
          if (key) fd.append("project_key", key);
          if (name) fd.append("project_name", name);
          result = await apiForm("/api/v1/import/file", fd);
        } else {
          const body = {
            filename: preview.filename,
            source: lastSource,
          };
          const key = keyInput.value.trim();
          const name = nameInput.value.trim();
          if (key) body.project_key = key;
          if (name) body.project_name = name;
          result = await api("/api/v1/import/file", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          });
        }
        status.textContent = `Kaydedildi · ${result.issues_found} sorun · Kapı: ${result.gate?.status_tr || "—"}`;
        status.className = "upload-status ok";
        state.projects = await api("/api/v1/projects/overview");
        location.hash = `#/project/${encodeURIComponent(result.project_key)}`;
      } catch (err) {
        status.textContent = "Kayıt hatası: " + err.message;
        status.className = "upload-status err";
      }
    };
  }

  document.getElementById("upload-submit").onclick = async () => {
    const status = document.getElementById("upload-status");
    const file = pickedFile || fileInput.files?.[0];

    if (!file) {
      status.textContent = "Önce bir dosya seç veya sürükle.";
      status.className = "upload-status err";
      return;
    }

    status.textContent = "Analiz ediliyor...";
    status.className = "upload-status";
    try {
      let preview;
      if (isZipFile(file)) {
        const fd = new FormData();
        fd.append("file", file);
        const key = keyInput.value.trim();
        const name = nameInput.value.trim();
        if (key) fd.append("project_key", key);
        if (name) fd.append("project_name", name);
        preview = await apiForm("/api/v1/import/preview", fd);
        lastSource = "";
      } else {
        lastSource = normalizeSource(await file.text());
        const body = {
          filename: file.name,
          source: lastSource,
        };
        const key = keyInput.value.trim();
        const name = nameInput.value.trim();
        if (key) body.project_key = key;
        if (name) body.project_name = name;
        preview = await api("/api/v1/import/preview", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
      }
      status.textContent = formatUploadStatus(preview);
      status.className = "upload-status ok";
      showUploadPreview(preview);
    } catch (err) {
      status.textContent = "Hata: " + err.message;
      status.className = "upload-status err";
    }
  };
}

function formatUploadStatus(preview) {
  const issues = preview.issues || [];
  const b = issueBreakdown(issues);
  const gate = preview.gate?.status_tr || preview.gate?.status || "—";
  if (b.blocker + b.critical + b.vuln > 0) {
    return `Analiz tamamlandı · ${issues.length} sorun · ${b.blocker + b.critical} kritik · Kapı: ${gate}`;
  }
  return `Analiz tamamlandı · ${issues.length} uyarı · Kapı: ${gate}`;
}

async function renderHistory() {
  setPage("Geçmiş");
  renderNav("history");

  const root = document.getElementById("app-root");
  root.innerHTML = `<div class="empty"><p>Yükleniyor...</p></div>`;

  try {
    const history = await api("/api/v1/history");
    if (!history.length) {
      root.innerHTML = `
        <div class="empty empty-compact">
          <p>Henüz kayıt yok.</p>
          <a href="#/upload" class="btn primary">Dosya yükle</a>
        </div>`;
      return;
    }

    root.innerHTML = `
      <div class="history-table-wrap">
        <table class="history-table">
          <thead>
            <tr>
              <th>Tarih</th>
              <th>Proje</th>
              <th>Bulunan</th>
              <th>Yeni</th>
              <th>Açık</th>
              <th>Kapı</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            ${history.map(h => `
              <tr>
                <td>${escapeHtml(formatWhen(h.created_at))}</td>
                <td>
                  <strong>${escapeHtml(h.project_name || h.project_key)}</strong>
                  <div class="meta">${escapeHtml(h.project_key)}</div>
                </td>
                <td>${h.issues_found}</td>
                <td>${h.issues_new}</td>
                <td>${h.issues_open}</td>
                <td><span class="gate-pill ${(h.gate_status || "PASS").toLowerCase()}">${escapeHtml(h.gate_status_tr || trGate(h.gate_status))}</span></td>
                <td><a class="link-btn" href="#/project/${encodeURIComponent(h.project_key)}">Detay</a></td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>`;
  } catch (err) {
    root.innerHTML = `<div class="empty"><h3>Hata</h3><p>${escapeHtml(err.message)}</p></div>`;
  }
}

async function renderProjects() {
  setPage("Projeler");
  const root = document.getElementById("app-root");
  root.innerHTML = `<div class="empty"><p>Yükleniyor...</p></div>`;
  try {
    state.projects = await api("/api/v1/projects/overview");
    renderNav("projects");
    if (!state.projects.length) {
      root.innerHTML = `
        <div class="empty empty-compact">
          <p>Henüz proje yok.</p>
          <a href="#/upload" class="btn primary">Dosya yükle</a>
        </div>`;
      return;
    }
    state.projectSearch = state.projectSearch || "";
    const renderGrid = () => {
      const filtered = filterProjects(state.projects, state.projectSearch);
      const grid = document.getElementById("project-grid");
      const empty = document.getElementById("project-search-empty");
      if (!grid) return;
      if (!filtered.length) {
        grid.innerHTML = "";
        if (empty) empty.classList.remove("hidden");
        return;
      }
      if (empty) empty.classList.add("hidden");
      grid.innerHTML = filtered.map(p => `
      <div class="project-card ${state.selectedProjects?.has(p.key) ? "selected" : ""}">
        <label class="project-select" title="Seç">
          <input type="checkbox" data-select-project="${escapeHtml(p.key)}" ${state.selectedProjects?.has(p.key) ? "checked" : ""}>
        </label>
        <div class="card-click" data-project="${escapeHtml(p.key)}">
          <div class="card-top">
            <h3>${escapeHtml(p.name)}</h3>
            ${p.gate ? `<span class="gate-pill ${p.gate.status.toLowerCase()}">${escapeHtml(p.gate.status_tr || trGate(p.gate.status))}</span>` : ""}
          </div>
          <div class="count-label">açık sorun</div>
          <div class="count">${p.open_issues}</div>
          <div class="meta">${p.vulnerabilities} güvenlik açığı · ${p.bugs} hata · ${p.code_smells} kod kokusu</div>
          <div class="meta project-key">${escapeHtml(p.key)}</div>
        </div>
        <div class="card-actions">
          <button type="button" class="btn ghost btn-sm" data-export-json="${escapeHtml(p.key)}">JSON</button>
          <button type="button" class="btn ghost btn-sm" data-export-html="${escapeHtml(p.key)}">HTML</button>
          <button type="button" class="btn ghost btn-sm" data-delete-project="${escapeHtml(p.key)}" data-project-name="${escapeHtml(p.name)}">Sil</button>
        </div>
      </div>`).join("");
      bindProjectCardActions(grid);
      updateBulkDeleteButton();
    };
    root.innerHTML = `
      <div class="page-actions">
        <a href="#/upload" class="btn primary">+ Dosya Yükle</a>
        <a href="#/history" class="btn secondary">Geçmişi Gör</a>
        <button type="button" class="btn ghost" id="bulk-select-all">Tümünü seç</button>
        <button type="button" class="btn ghost" id="bulk-clear">Seçimi temizle</button>
        <button type="button" class="btn ghost" id="bulk-delete" disabled>Seçilenleri sil</button>
      </div>
      <div class="project-search-bar">
        <input type="search" id="project-search" placeholder="Proje adı veya kodu ara…" value="${escapeHtml(state.projectSearch)}" autocomplete="off">
        <span class="project-search-count" id="project-search-count">${state.projects.length} proje</span>
      </div>
      <div class="empty hidden" id="project-search-empty">
        <p>Aramanızla eşleşen proje yok.</p>
      </div>
      <div class="project-grid" id="project-grid"></div>`;
    renderGrid();
    document.getElementById("bulk-delete")?.addEventListener("click", bulkDeleteProjects);
    document.getElementById("bulk-select-all")?.addEventListener("click", () => {
      const filtered = filterProjects(state.projects, state.projectSearch);
      state.selectedProjects = new Set(filtered.map(p => p.key));
      renderGrid();
    });
    document.getElementById("bulk-clear")?.addEventListener("click", () => {
      state.selectedProjects = new Set();
      renderGrid();
    });
    document.getElementById("project-search")?.addEventListener("input", e => {
      state.projectSearch = e.target.value;
      const filtered = filterProjects(state.projects, state.projectSearch);
      const countEl = document.getElementById("project-search-count");
      if (countEl) countEl.textContent = `${filtered.length} / ${state.projects.length} proje`;
      renderGrid();
    });
  } catch (err) {
    root.innerHTML = `<div class="empty"><h3>Hata</h3><p>${escapeHtml(err.message)}</p></div>`;
  }
}

function bindProjectCardActions(container) {
  container.querySelectorAll("[data-select-project]").forEach(box => {
    box.onclick = e => e.stopPropagation();
    box.onchange = e => {
      e.stopPropagation();
      if (!state.selectedProjects) state.selectedProjects = new Set();
      if (box.checked) state.selectedProjects.add(box.dataset.selectProject);
      else state.selectedProjects.delete(box.dataset.selectProject);
      box.closest(".project-card")?.classList.toggle("selected", box.checked);
      updateBulkDeleteButton();
    };
  });
  container.querySelectorAll("[data-project]").forEach(el => {
    el.onclick = () => { location.hash = `#/project/${encodeURIComponent(el.dataset.project)}`; };
  });
  container.querySelectorAll("[data-export-json]").forEach(btn => {
    btn.onclick = e => { e.stopPropagation(); downloadProjectExport(btn.dataset.exportJson, "json"); };
  });
  container.querySelectorAll("[data-export-html]").forEach(btn => {
    btn.onclick = e => { e.stopPropagation(); downloadProjectExport(btn.dataset.exportHtml, "html"); };
  });
  container.querySelectorAll("[data-delete-project]").forEach(btn => {
    btn.onclick = e => {
      e.stopPropagation();
      deleteProject(btn.dataset.deleteProject, btn.dataset.projectName);
    };
  });
}

async function renderProject(key) {
  const root = document.getElementById("app-root");
  root.innerHTML = `<div class="empty"><span class="loading"></span>Yükleniyor...</div>`;
  state.filter = "ALL";
  state.typeFilter = "ALL";
  state.statusFilter = "OPEN";
  state.projectKey = key;
  try {
    if (!state.projects.length) state.projects = await api("/api/v1/projects/overview");
    renderNav("projects");
    const [issues, overview, history, sourceInfo] = await Promise.all([
      api(`/api/v1/projects/${encodeURIComponent(key)}/issues`),
      api(`/api/v1/projects/${encodeURIComponent(key)}/overview`),
      api(`/api/v1/projects/${encodeURIComponent(key)}/history`),
      api(`/api/v1/projects/${encodeURIComponent(key)}/source`).catch(() => ({ available: false })),
    ]);
    state.issues = issues;
    state.projectGate = overview.gate;
    state.selectedId = filteredIssues(issues)[0]?.id || null;

    const projectName = overview.name || key;
    const files = [...new Set(issues.map(i => i.file).filter(Boolean))];
    const hasSource = sourceInfo.available && sourceInfo.source;
    const isArchive = !!sourceInfo.archive && sourceInfo.files?.length;

    setPage(projectName, key, `
      <button type="button" class="btn secondary" id="export-json">JSON</button>
      <button type="button" class="btn secondary" id="export-html">HTML</button>
      <button type="button" class="btn secondary" id="export-preview">PDF</button>
      ${hasSource ? `<button type="button" class="btn secondary" id="project-rescan">Tara</button>` : ""}
      <button type="button" class="btn ghost" id="project-delete">Sil</button>
      <a href="#/projects" class="btn ghost">←</a>
    `);

    if (hasSource) {
      state.editorMode = "project";
      const activeFile = sourceInfo.active_file || sourceInfo.files?.[0]?.filename || files[0] || "";
      const activeIssues = isArchive ? issues.filter(i => i.file === activeFile) : issues;
      state.projectSource = {
        source: normalizeSource(sourceInfo.source),
        issues,
        selectedLine: activeIssues[0]?.line || issues[0]?.line || null,
        filename: activeFile || sourceInfo.filename || "",
        archive: isArchive,
        files: sourceInfo.files || [],
        activeFile,
      };
    }

    root.innerHTML = `
      ${`<div class="project-gate">${renderGateBadge(overview.gate, issues.filter(i => i.status !== "CLOSED"))}</div>`}
      ${renderSeverityStats(issues)}
      ${renderMeasuresGrid(overview.measures)}
      ${renderGateHistory(history)}
      ${hasSource ? `
      <div class="project-code-panel">
        <div class="panel-title">${escapeHtml(state.projectSource.filename)}</div>
        ${isArchive ? renderArchiveFilePicker(state.projectSource, "project-file-select") : ""}
        <div class="editor-shell code-overview">
          <div class="gutter" id="gutter"></div>
          <div class="code-editor">
            <div class="code-highlights" id="code-highlights" aria-hidden="true"></div>
            <textarea id="code-source" spellcheck="false" readonly></textarea>
          </div>
        </div>
      </div>` : ""}
      <div class="filters" id="issue-filters"></div>
      <div class="filter-meta" id="filter-meta"></div>
      <div class="workspace">
        <div class="issue-list" id="issue-list"></div>
        <div class="fix-panel" id="fix-panel"></div>
      </div>`;
    if (hasSource) {
      mountCodeEditor({ readonly: true });
      bindArchiveFilePicker(state.projectSource, "project-file-select");
    }
    renderIssueFilters();
    renderStoredIssueList();
    renderStoredFixPanel();
    document.getElementById("project-rescan")?.addEventListener("click", () => rescanProject(key));
    document.getElementById("export-json")?.addEventListener("click", () => downloadProjectExport(key, "json"));
    document.getElementById("export-html")?.addEventListener("click", () => downloadProjectExport(key, "html"));
    document.getElementById("export-preview")?.addEventListener("click", () => previewProjectExport(key));
    document.getElementById("project-delete")?.addEventListener("click", () => deleteProject(key, projectName));
  } catch (err) {
    root.innerHTML = `<div class="empty"><h3>Hata</h3><p>${escapeHtml(err.message)}</p></div>`;
  }
}

function renderIssueFilters() {
  const el = document.getElementById("issue-filters");
  const meta = document.getElementById("filter-meta");
  if (!el) return;

  const severities = [
    { id: "ALL", label: "Tümü" },
    { id: "BLOCKER", label: trSeverity("BLOCKER") },
    { id: "CRITICAL", label: trSeverity("CRITICAL") },
    { id: "MAJOR", label: trSeverity("MAJOR") },
    { id: "MINOR", label: trSeverity("MINOR") },
  ];
  const types = [
    { id: "ALL", label: "Tüm türler" },
    { id: "BUG", label: trType("BUG") },
    { id: "VULNERABILITY", label: trType("VULNERABILITY") },
    { id: "CODE_SMELL", label: trType("CODE_SMELL") },
  ];
  const statuses = [
    { id: "OPEN", label: "Açık" },
    { id: "SUPPRESSED", label: "Yanlış alarm" },
    { id: "ALL", label: "Tümü" },
  ];

  el.innerHTML = `
    <div class="filter-group">
      <span class="filter-label">Durum:</span>
      ${statuses.map(s => `
        <button class="chip ${state.statusFilter === s.id ? "active" : ""}" data-status="${s.id}">${s.label}</button>
      `).join("")}
    </div>
    <div class="filter-group">
      <span class="filter-label">Önem:</span>
      ${severities.map(s => `
        <button class="chip ${state.filter === s.id ? "active" : ""}" data-filter="${s.id}">${s.label}</button>
      `).join("")}
    </div>
    <div class="filter-group">
      <span class="filter-label">Tür:</span>
      ${types.map(t => `
        <button class="chip ${state.typeFilter === t.id ? "active" : ""}" data-type="${t.id}">${t.label}</button>
      `).join("")}
    </div>`;

  el.querySelectorAll("[data-status]").forEach(btn => {
    btn.onclick = () => {
      state.statusFilter = btn.dataset.status;
      const visible = filteredIssues(state.issues);
      if (!visible.some(i => i.id === state.selectedId)) {
        state.selectedId = visible[0]?.id || null;
      }
      renderIssueFilters();
      renderStoredIssueList();
      renderStoredFixPanel();
    };
  });
  el.querySelectorAll("[data-filter]").forEach(btn => {
    btn.onclick = () => {
      state.filter = btn.dataset.filter;
      const visible = filteredIssues(state.issues);
      if (!visible.some(i => i.id === state.selectedId)) {
        state.selectedId = visible[0]?.id || null;
      }
      renderIssueFilters();
      renderStoredIssueList();
      renderStoredFixPanel();
    };
  });
  el.querySelectorAll("[data-type]").forEach(btn => {
    btn.onclick = () => {
      state.typeFilter = btn.dataset.type;
      const visible = filteredIssues(state.issues);
      if (!visible.some(i => i.id === state.selectedId)) {
        state.selectedId = visible[0]?.id || null;
      }
      renderIssueFilters();
      renderStoredIssueList();
      renderStoredFixPanel();
    };
  });

  if (meta) {
    const visible = filteredIssues(state.issues);
    meta.textContent = `${visible.length} / ${state.issues.length} sorun gösteriliyor`;
  }
}

function renderStoredIssueList() {
  const el = document.getElementById("issue-list");
  const issues = filteredIssues(state.issues);
  if (!issues.length) {
    el.innerHTML = `<div class="empty"><p>Bu filtreye uygun sorun yok</p></div>`;
    return;
  }
  el.innerHTML = issues.map(issue => `
    <div class="issue-item ${issue.id === state.selectedId ? "active" : ""} ${issue.resolution === "FALSE_POSITIVE" || issue.resolution === "WONTFIX" ? "suppressed" : ""}" data-id="${issue.id}">
      <div class="title">
        <span class="badge ${issue.severity}">${trSeverity(issue.severity)}</span>
        ${issue.resolution === "FALSE_POSITIVE" || issue.resolution === "WONTFIX"
          ? `<span class="resolution-pill">${escapeHtml(trResolution(issue.resolution))}</span>` : ""}
        ${escapeHtml(issue.message)}
      </div>
      <div class="meta">${escapeHtml(issue.file)}:${issue.line}</div>
    </div>
  `).join("");
  el.querySelectorAll(".issue-item").forEach(item => {
    item.onclick = () => {
      state.selectedId = item.dataset.id;
      const issue = state.issues.find(i => i.id === state.selectedId);
      if (issue && state.editorMode === "project") {
        if (state.projectSource.archive && issue.file && issue.file !== state.projectSource.activeFile) {
          setActiveArchiveFile(state.projectSource, issue.file);
          state.selectedId = issue.id;
        }
        state.projectSource.selectedLine = issue.line;
        syncEditorView();
      }
      renderStoredIssueList();
      renderStoredFixPanel();
    };
  });
}

function renderStoredFixPanel() {
  const panel = document.getElementById("fix-panel");
  const issue = state.issues.find(i => i.id === state.selectedId);
  if (!issue) {
    panel.innerHTML = `<div class="empty"><p>Sorun seç</p></div>`;
    return;
  }
  if (state.editorMode === "project") {
    state.projectSource.selectedLine = issue.line;
    syncEditorView();
  }
  const ctx = state.projectSource.source ? state.projectSource : null;
  const codeLine = issueCodeLine(issue, ctx?.source || "");
  panel.innerHTML = `
    <div class="issue-head">
      <span class="badge ${issue.severity}">${trSeverity(issue.severity)}</span>
      <span class="type-tag">${trType(issue.type)}</span>
      <span class="file-tag">${escapeHtml(issue.file)}:${issue.line}</span>
    </div>
    <h3>${escapeHtml(issue.message)}</h3>
    <div class="code-head"><span>Mevcut kod (satır ${issue.line})</span></div>
    <pre class="code-block bad-line">${escapeHtml(codeLine || issue.snippet || "Kod parçası mevcut değil")}</pre>
    <div class="code-head"><span>Önerilen düzeltme</span><button class="btn primary" id="copy-fix">Kopyala</button></div>
    <pre class="code-block fix">${escapeHtml(issue.fix_suggestion || fixFallback(issue))}</pre>
    ${renderAIHelpUI(issueAIKey(issue))}
    ${renderResolutionActions(issue)}`;
  document.getElementById("copy-fix").onclick = () => navigator.clipboard.writeText(issue.fix_suggestion || fixFallback(issue));
  bindResolutionActions(issue);
  bindAIHelp(issue, codeLine);
}

function escapeHtml(v) {
  return String(v).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
}

async function checkHealth() {
  const el = document.getElementById("server-status");
  const dot = document.getElementById("status-dot");
  const aiPill = document.getElementById("ai-status-pill");
  if (!el || !dot) return;

  try {
    const res = await fetch("/api/health");
    if (!res.ok) throw new Error();
    const data = await res.json();
    syncAIStatus(data.ai);
    dot.classList.add("ok");
    el.textContent = "Sunucu çalışıyor";
    if (state.aiActive) {
      dot.classList.add("ai-ok");
      if (aiPill) {
        aiPill.classList.remove("hidden");
        aiPill.textContent = aiProviderLabel();
      }
    } else {
      dot.classList.remove("ai-ok");
      if (aiPill) {
        aiPill.classList.remove("hidden");
        aiPill.textContent = "YZ kapalı";
      }
    }
  } catch {
    el.textContent = "Sunucu bağlantısı yok";
    dot.classList.remove("ok", "ai-ok");
    if (aiPill) aiPill.classList.add("hidden");
  }
}

window.addEventListener("hashchange", () => route());

checkHealth();
ensureToken().then(() => {
  api("/api/v1/projects/overview").then(p => { state.projects = p; renderNav("overview"); }).catch(() => {});
  route();
}).catch(async () => {
  try {
    const cfg = await fetch("/api/public/config").then(r => r.json());
    if (cfg.auth_required) {
      window.location.href = "/login?next=" + encodeURIComponent(location.pathname + location.hash);
      return;
    }
  } catch { /* ignore */ }
  document.getElementById("app-root").innerHTML =
    `<div class="empty"><h3>Bağlantı hatası</h3><p>Sunucu çalışıyor mu? server.bat ile yeniden başlatın.</p></div>`;
});
