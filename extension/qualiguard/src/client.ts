export type IssueSeverity = "BLOCKER" | "CRITICAL" | "MAJOR" | "MINOR" | "INFO";

export interface QualiGuardIssue {
  rule_key: string;
  severity: IssueSeverity;
  type: string;
  message: string;
  file: string;
  line: number;
  column?: number;
  fix_suggestion?: string;
}

export interface QualiGuardReport {
  issues: QualiGuardIssue[];
  gate?: { status: string; status_tr?: string };
}

interface PublicConfig {
  auth_required?: boolean;
}

export class QualiGuardClient {
  constructor(
    private baseUrl: string,
    private token: string,
  ) {
    this.baseUrl = baseUrl.replace(/\/+$/, "");
  }

  setToken(token: string) {
    this.token = token;
  }

  getToken(): string {
    return this.token;
  }

  async health(): Promise<boolean> {
    try {
      const res = await fetch(`${this.baseUrl}/api/health`, { method: "GET" });
      return res.ok;
    } catch {
      return false;
    }
  }

  async publicConfig(): Promise<PublicConfig> {
    try {
      const res = await fetch(`${this.baseUrl}/api/public/config`, { method: "GET" });
      if (!res.ok) {
        return {};
      }
      return (await res.json()) as PublicConfig;
    } catch {
      return {};
    }
  }

  async bootstrapToken(): Promise<string> {
    const res = await fetch(`${this.baseUrl}/api/bootstrap`, { method: "GET" });
    if (!res.ok) {
      const cfg = await this.publicConfig();
      if (cfg.auth_required) {
        throw new Error(
          "Panel şifresi gerekli. Ayarlar → qualiguard.token alanına API token yapıştırın.",
        );
      }
      throw new Error(`Bootstrap başarısız (${res.status}). Sunucu çalışıyor mu?`);
    }
    const data = (await res.json()) as { token?: string };
    if (!data.token) {
      throw new Error("API token alınamadı");
    }
    return data.token;
  }

  async loginWithPassword(password: string): Promise<string> {
    const res = await fetch(`${this.baseUrl}/api/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    });
    if (!res.ok) {
      const body = (await res.json().catch(() => ({}))) as { error?: string };
      throw new Error(body.error || `Giriş başarısız (${res.status})`);
    }
    const data = (await res.json()) as { token?: string };
    if (!data.token) {
      throw new Error("Token alınamadı");
    }
    this.token = data.token;
    return data.token;
  }

  async ensureToken(): Promise<string> {
    if (this.token?.startsWith("qg_")) {
      return this.token;
    }
    this.token = await this.bootstrapToken();
    return this.token;
  }

  async analyze(filename: string, source: string): Promise<QualiGuardReport> {
    const token = await this.ensureToken();
    const res = await fetch(`${this.baseUrl}/api/v1/analyze/code`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ source, filename }),
    });
    if (res.status === 401) {
      this.token = "";
      throw new Error("Yetkisiz — token geçersiz. qualiguard.connect ile yeniden bağlanın.");
    }
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `Analiz hatası (${res.status})`);
    }
    return (await res.json()) as QualiGuardReport;
  }
}

export const SUPPORTED_EXT = new Set([
  ".py",
  ".js",
  ".jsx",
  ".ts",
  ".tsx",
  ".go",
  ".java",
  ".cs",
]);

export function isSupportedFile(filePath: string): boolean {
  const lower = filePath.toLowerCase();
  for (const ext of SUPPORTED_EXT) {
    if (lower.endsWith(ext)) {
      return true;
    }
  }
  return false;
}
