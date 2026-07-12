# Yapay Zeka Entegrasyonu

QualiGuard'ın SonarQube'dan ayrıştığı ana nokta: **statik analiz + AI katmanı**.

---

## 1. Temel Fikir

SonarQube kuralları **pattern ve AST** ile çalışır — hızlı ama bağlamı anlamaz.

AI katmanı şunu ekler:
- Kodun **ne yapmaya çalıştığını** anlar
- Issue'ları **bağlama göre** önceliklendirir
- **False positive** filtreler
- **Düzeltme önerisi** üretir
- Doğal dille **özel kural** yazdırır

```
┌─────────────────────────────────────────────────────────┐
│                    QualiGuard Scan                     │
│                                                          │
│  Katman 1: Statik Kurallar (hızlı, ücretsiz, offline)   │
│  ─────────────────────────────────────────────────────  │
│  AST · Regex · Complexity · Duplication                  │
│  → Kesin bulgular, milisaniyeler                         │
│                                                          │
│  Katman 2: AI Analiz (opsiyonel, akıllı, bağlamsal)   │
│  ─────────────────────────────────────────────────────  │
│  LLM · RAG · Embedding · Semantic Security               │
│  → Derin bulgular, saniyeler, API maliyeti               │
│                                                          │
│  Katman 3: AI Assist (UX katmanı)                       │
│  ─────────────────────────────────────────────────────  │
│  Açıklama · Auto-fix · Önceliklendirme · Chat           │
│  → Developer experience                                  │
└─────────────────────────────────────────────────────────┘
```

**Kural:** Statik analiz her zaman çalışır. AI opsiyonel ama güçlü fark yaratıcı.

---

## 2. AI Kullanım Alanları (7 Modül)

### Modül 1: AI Issue Açıklama (En kolay başlangıç)

Statik kural bir issue buldu → AI bağlamda açıklar.

```
Statik:  python:sql-injection @ utils.py:42
AI ekler:
  ├── "Bu fonksiyon kullanıcı girdisini doğrudan SQL'e ekliyor"
  ├── "Saldırgan ' OR 1=1 -- ile tüm veritabanını okuyabilir"
  ├── "Bu endpoint /api/users public — risk: CRITICAL"
  └── Düzeltme: cursor.execute("SELECT * FROM users WHERE id = ?", (user_id,))
```

**Ne zaman çalışır:** Issue oluştuktan sonra, sunucu tarafında (async)  
**Maliyet:** Issue başına ~500 token → ucuz  
**Faz:** AI-1 (Faz 2 ile birlikte)

---

### Modül 2: AI False Positive Filtresi

Statik kurallar gürültü yapar. AI triyaj yapar:

```
Statik:  47 issue buldu
AI:      12'si muhtemelen false positive (confidence > 0.8)
         35'i gerçek issue
         8'i yüksek öncelikli
```

**Nasıl çalışır:**
```python
prompt = f"""
Statik analiz aracı şu issue'yu buldu:
- Kural: {rule_key}
- Dosya: {file_path}:{line}
- Mesaj: {message}
- Kod snippet:
{code_snippet}

Bu gerçek bir sorun mu yoksa false positive mi?
JSON döndür: {{"is_real": bool, "confidence": 0-1, "reason": "..."}}
"""
```

**Öğrenme:** Takım "false positive" işaretlediğinde → fine-tune veya few-shot örnek olarak sakla.

**Faz:** AI-2

---

### Modül 3: AI Semantic Security Scan

Statik kurallar bilinen pattern'leri yakalar. AI **mantık hatalarını** yakalar:

| Statik yakalar | AI yakalar |
|----------------|------------|
| `eval(user_input)` | Race condition in auth flow |
| Hardcoded password regex | Business logic bypass |
| SQL string concat | IDOR — user A, user B'nin datasına erişiyor |
| `os.system()` | Timing attack in token comparison |
| Unused variable | Insecure deserialization chain |

**Nasıl çalışır:**
```
1. Dosyayı/fonksiyonu parse et
2. İlgili context topla (imports, callers, data flow)
3. LLM'e gönder: "Bu kodda güvenlik açığı var mı?"
4. Structured JSON issue listesi al
5. Statik issue'larla merge et (duplicate kontrol)
```

**Prompt stratejisi:**
```yaml
system: |
  Sen bir güvenlik analistisin. Verilen kodu incele.
  SADECE gerçek güvenlik risklerini raporla.
  JSON formatında döndür, spekülasyon yapma.

user: |
  Language: {lang}
  File: {path}
  Function: {func_name}
  
  ```{lang}
  {code}
  ```
  
  Callers: {callers}
  Data inputs: {inputs}
```

**Maliyet kontrolü:**
- Sadece `security`-tag'li dosyalar
- Sadece yeni/değişen kod (PR diff)
- Sadece CRITICAL+ statik issue olan dosyalar
- Cache: aynı hash → tekrar sorma

**Faz:** AI-3

---

### Modül 4: AI Auto-Fix

Issue bulundu → AI düzeltme patch'i üretir:

```
Issue: python:unused-import @ app.py:3
  import os  ← kullanılmıyor

AI Fix:
  --- a/app.py
  +++ b/app.py
  @@ -1,5 +1,4 @@
   import sys
  -import os
   from flask import Flask
```

**CLI kullanımı:**
```bash
qg fix --issue-id abc123          # Tek issue düzelt
qg fix --auto --severity MINOR    # Tüm minor issue'ları düzelt
qg fix --dry-run                  # Patch göster, uygulama
```

**Güvenlik:**
- Auto-fix default kapalı
- CRITICAL/BLOCKER için manuel onay zorunlu
- Patch review UI'da gösterilir
- `git apply` ile uygulanır

**Faz:** AI-4

---

### Modül 5: Doğal Dil ile Özel Kural

Developer veya security team doğal dille kural yazar:

```
Kullanıcı: "Kullanıcı girdisi olan tüm dosyalarda 
            password kelimesi geçen string literal'leri bul"

AI → Custom rule oluşturur:
  - AST pattern VEYA
  - Semantic search prompt VEYA
  - Regex + context filter
```

**UI:**
```
┌─────────────────────────────────────────────┐
│  🔍 Yeni Kural Oluştur                      │
│                                              │
│  "API endpoint'lerinde rate limiting yoksa  │
│   uyar"                                      │
│                              [Oluştur →]    │
│                                              │
│  Önizleme:                                   │
│  Key: custom:no-rate-limit                   │
│  Type: SECURITY_HOTSPOT                      │
│  Detection: ast_visitor + semantic           │
└─────────────────────────────────────────────┘
```

**Faz:** AI-5

---

### Modül 6: AI PR Review Özeti

PR açıldığında otomatik özet:

```markdown
## QualiGuard AI Review — PR #42

### Özet
Bu PR auth modülünü refactor ediyor. 3 güvenlik riski, 
2 code smell tespit edildi.

### Kritik Bulgular
1. **IDOR Risk** — `get_user()` endpoint'inde user_id 
   session'dan değil URL'den alınıyor (src/auth.py:87)
2. **Token Expiry** — JWT refresh token süresiz (src/auth.py:120)

### Kalite Değişimi
| Metrik | Önce | Sonra |
|--------|------|-------|
| Coverage | 78% | 82% ✅ |
| Complexity | 340 | 355 ⚠️ |
| Security Rating | B | C ❌ |

### Öneri
PR merge edilmeden önce #1 IDOR riski düzeltilmeli.
```

**Faz:** AI-4 (CI entegrasyonu ile)

---

### Modül 7: Codebase Chat (RAG)

Developer soru sorar, AI codebase'e bakarak cevaplar:

```
S: "Projede kaç yerde SQL injection riski var?"
C: 3 dosyada 5 nokta tespit edildi:
   1. src/db/queries.py:42 — f-string SQL
   2. src/api/search.py:18 — concat query
   ...

S: "En riskli endpoint hangisi?"
C: POST /api/admin/users — auth bypass + SQL injection 
   kombinasyonu. Öncelik: P0
```

**Teknik:**
```
Codebase → Chunk → Embed → Vector DB (pgvector / sqlite-vec)
Soru → Embed → Similarity search → Top-K chunks → LLM answer
```

**Faz:** AI-6

---

## 3. Mimari: AI Katmanı Nereye Oturur?

```
┌──────────────────────────────────────────────────────────┐
│                    QualiGuard Server                      │
│                                                           │
│  ┌─────────────┐    ┌─────────────────────────────────┐  │
│  │ Static      │    │         AI Engine               │  │
│  │ Processor   │───→│                                 │  │
│  │ (mevcut)    │    │  ┌───────────┐ ┌────────────┐  │  │
│  └─────────────┘    │  │ Triage    │ │ Semantic   │  │  │
│                     │  │ Service   │ │ Scanner    │  │  │
│                     │  └───────────┘ └────────────┘  │  │
│                     │  ┌───────────┐ ┌────────────┐  │  │
│                     │  │ Explainer │ │ Auto-Fix   │  │  │
│                     │  └───────────┘ └────────────┘  │  │
│                     │  ┌───────────┐ ┌────────────┐  │  │
│                     │  │ NL Rules  │ │ RAG Chat   │  │  │
│                     │  └───────────┘ └────────────┘  │  │
│                     └──────────┬──────────────────────┘  │
│                                │                          │
│                     ┌──────────┴──────────┐                │
│                     │   LLM Provider     │                │
│                     │   (pluggable)      │                │
│                     └──────────┬──────────┘                │
└────────────────────────────────┼──────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
        ┌──────────┐     ┌──────────┐     ┌──────────┐
        │ OpenAI   │     │ Anthropic│     │ Ollama   │
        │ GPT-4o   │     │ Claude   │     │ (local)  │
        └──────────┘     └──────────┘     └──────────┘
```

**CLI tarafında AI yok** (MVP'de) — AI sunucuda çalışır.  
İstisna: `qg fix` komutu lokal LLM (Ollama) destekleyebilir.

---

## 4. LLM Provider Abstraction

```go
// internal/ai/provider.go
type LLMProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    Name() string
}

// Implementations
type OpenAIProvider struct { apiKey, model string }
type AnthropicProvider struct { apiKey, model string }
type OllamaProvider struct { baseURL, model string }  // local, free
type AzureOpenAIProvider struct { endpoint, apiKey, deployment string }
```

### Config
```yaml
# qualiguard.yaml (server)
ai:
  enabled: true
  provider: ollama          # openai | anthropic | ollama | azure
  model: llama3.1:8b         # or gpt-4o-mini, claude-3-haiku
  
  # Maliyet kontrolü
  max_tokens_per_scan: 50000
  max_issues_to_explain: 20
  semantic_scan: true
  semantic_scan_scope: new_code  # new_code | all | security_files
  
  # Auto-fix
  autofix:
    enabled: false
    auto_apply: false         # true = tehlikeli
    max_severity: MINOR       # sadece MINOR ve altı otomatik
  
  # RAG
  rag:
    enabled: false
    embedding_model: text-embedding-3-small
    chunk_size: 512
```

---

## 5. Maliyet ve Performans Kontrolü

AI pahalı olabilir — kontrol mekanizmaları:

### Token Budget
```go
type TokenBudget struct {
    MaxPerScan    int  // 50K token/scan
    MaxPerIssue   int  // 2K token/issue
    MaxPerDay     int  // 1M token/gün (org limit)
    CurrentUsed   int
}
```

### Akıllı Tetikleme (ne zaman AI çağrılır)
| Durum | AI çağrılır mı? |
|-------|-----------------|
| INFO severity issue | ❌ Hayır |
| MINOR code smell | ❌ (sadece açıklama istenirse) |
| MAJOR bug | ✅ Triage |
| CRITICAL/BLOCKER | ✅ Triage + Semantic + Explain |
| SECURITY issue | ✅ Full AI pipeline |
| Kullanıcı "Explain" tıkladı | ✅ Explain |
| PR açıldı | ✅ Summary (new code only) |
| Aynı fingerprint (cache hit) | ❌ Cache'den dön |

### Cache
```
cache_key = hash(rule_key + file_path + code_snippet + model_version)
TTL = 7 gün (kod değişince zaten yeni hash)
```

### Local LLM (Ollama) — maliyetsiz alternatif
```bash
# Tamamen offline, ücretsiz
ollama pull llama3.1:8b
ollama pull codellama:13b  # kod analizi için daha iyi

# qualiguard.yaml
ai:
  provider: ollama
  model: codellama:13b
```
Kalite cloud'tan düşük ama gizlilik + maliyet avantajı büyük.

---

## 6. AI Issue Formatı (Statik + AI Birleşik)

```json
{
  "rule_key": "python:sql-injection",
  "severity": "BLOCKER",
  "type": "VULNERABILITY",
  "source": "static",
  "message": "SQL query built with string formatting",
  "file": "src/db.py",
  "line": 42,
  
  "ai": {
    "enabled": true,
    "triage": {
      "is_real": true,
      "confidence": 0.95,
      "reason": "user_id parameter comes from request.args without validation"
    },
    "explanation": "Bu endpoint public. Saldırgan user_id parametresini manipüle ederek başka kullanıcıların verilerine erişebilir.",
    "impact": "CRITICAL — tüm kullanıcı veritabanı okunabilir",
    "fix_suggestion": "cursor.execute('SELECT * FROM users WHERE id = ?', (validated_id,))",
    "fix_patch": "--- a/src/db.py\n+++ b/src/db.py\n...",
    "cwe": ["CWE-89"],
    "owasp": ["A03:2021 Injection"],
    "priority_score": 98
  }
}
```

---

## 7. SonarQube AI vs QualiGuard AI

| Özellik | SonarQube | QualiGuard AI |
|---------|-----------|---------------|
| AI CodeFix | Enterprise, ücretli | Açık kaynak, opsiyonel |
| AI açıklama | Sınırlı | Her issue için |
| False positive AI | Yok | Triage modülü |
| Semantic security | Yok | AI-3 modülü |
| Doğal dil kural | Yok | AI-5 modülü |
| Local LLM | Yok | Ollama desteği |
| Codebase chat | Yok | RAG modülü |
| PR AI summary | Sınırlı | Tam özet |
| Türkçe açıklama | Yok | Prompt ile desteklenir |

**QualiGuard farkı:** AI baştan mimariye gömülü, enterprise duvarı yok, local LLM seçeneği var.

---

## 8. Gizlilik ve Güvenlik

Kod LLM'e gönderilirken dikkat:

```
1. Sensitive file exclusion
   - .env, credentials, secrets otomatik filtre
   - qualiguard.yaml → ai.exclude_patterns

2. PII redaction
   - Email, IP, token pattern'leri maskele
   - Göndermeden önce: replace(regex, "***")

3. Self-hosted seçeneği
   - Ollama = kod dışarı çıkmaz
   - Azure OpenAI = enterprise data residency

4. Opt-in
   - ai.enabled: false → default kapalı
   - Proje bazlı AI izni

5. Audit log
   - Hangi kod parçası hangi LLM'e gitti → log
```

---

## 9. Uygulama Fazları

```
AI-1  Issue Açıklama          ← Faz 2 ile birlikte (kolay, etkili)
AI-2  False Positive Triage  ← Faz 3
AI-3  Semantic Security       ← Faz 4
AI-4  Auto-Fix + PR Summary   ← Faz 4-5
AI-5  Doğal Dil Kurallar      ← Faz 5
AI-6  RAG Codebase Chat       ← Faz 5+
```

### AI-1 MVP (ilk yapılacak AI özelliği)

```go
// internal/ai/explainer.go
func ExplainIssue(ctx context.Context, issue Issue, snippet string) (*AIExplanation, error) {
    prompt := buildExplainPrompt(issue, snippet)
    resp, err := provider.Complete(ctx, CompletionRequest{
        System: "Sen bir kod kalitesi uzmanısın. Kısa ve net açıkla.",
        User:   prompt,
        MaxTokens: 500,
        ResponseFormat: "json",
    })
    // parse JSON → AIExplanation
}
```

**Tahmini efor:** 2-3 gün (OpenAI/Ollama provider + explain endpoint + UI butonu)

---

## 10. Örnek Kullanıcı Deneyimi

### Developer workflow
```
1. qg scan --upload                    → statik analiz
2. UI'da issue listesi                 → 23 issue
3. AI badge: "8 yüksek öncelikli"      → triage yapılmış
4. Issue'a tıkla → AI açıklama         → "Bu neden tehlikeli"
5. "Fix öner" butonu                   → patch göster
6. "Uygula" → git apply                → düzeltildi
7. qg scan --upload                    → gate PASS ✅
```

### Security team workflow
```
1. "API endpoint'lerinde auth bypass var mı?" → NL sorgu
2. AI semantic scan → 2 bulgu
3. Custom rule oluştur → kalıcı kural olarak kaydet
4. Quality gate'e ekle → otomatik kontrol
```

---

## 11. Sonraki Adım

1. Faz 1-2'de statik scanner'ı bitir (temel olmadan AI anlamsız)
2. AI-1 (Issue Explain) ile AI'yı tanıt
3. Ollama local LLM ile maliyetsiz demo yap
4. AI-2 (Triage) ile false positive sorununu çöz → **gerçek fark yaratır**

Sonraki dosya: `design/ai-mimari-diyagram.md`
