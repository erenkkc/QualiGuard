# AI Mimari Diyagramları

QualiGuard AI katmanı görsel akışları.

---

## 1. Hybrid Analiz Akışı (Statik + AI)

```mermaid
flowchart TD
    A[qg scan] --> B[Statik Rule Engine]
    B --> C{Issue bulundu?}
    C -->|Hayır| D[Rapor: temiz]
    C -->|Evet| E[AnalysisReport upload]
    E --> F[Server: Report Processor]
    F --> G[Issue Merge + DB]
    
    G --> H{AI enabled?}
    H -->|Hayır| I[Statik sonuç → UI]
    H -->|Evet| J[AI Pipeline]
    
    J --> K[Triage: FP filtresi]
    K --> L[Explain: açıklama üret]
    L --> M{Security issue?}
    M -->|Evet| N[Semantic Scan]
    M -->|Hayır| O[Priority Score]
    N --> O
    O --> P[AI-enriched sonuç → UI]
    
    P --> Q{Auto-fix enabled?}
    Q -->|Evet| R[Fix patch üret]
    Q -->|Hayır| S[Bitti]
    R --> S
```

---

## 2. AI Engine İç Yapısı

```mermaid
graph TB
    subgraph "AI Engine"
        ROUTER[AI Router / Orchestrator]
        
        subgraph "Services"
            TRIAGE[Triage Service]
            EXPLAIN[Explainer Service]
            SEMANTIC[Semantic Scanner]
            AUTOFIX[Auto-Fix Service]
            NLRULE[NL Rule Generator]
            RAG[RAG Chat Service]
        end
        
        subgraph "Shared"
            CACHE[(AI Cache)]
            BUDGET[Token Budget]
            REDACT[PII Redactor]
            PROMPT[Prompt Templates]
        end
        
        ROUTER --> TRIAGE
        ROUTER --> EXPLAIN
        ROUTER --> SEMANTIC
        ROUTER --> AUTOFIX
        ROUTER --> NLRULE
        ROUTER --> RAG
        
        TRIAGE --> CACHE
        EXPLAIN --> CACHE
        SEMANTIC --> REDACT
        AUTOFIX --> REDACT
        
        TRIAGE --> BUDGET
        EXPLAIN --> BUDGET
        SEMANTIC --> BUDGET
    end
    
    subgraph "LLM Providers"
        OPENAI[OpenAI]
        CLAUDE[Anthropic]
        OLLAMA[Ollama Local]
    end
    
    ROUTER --> OPENAI
    ROUTER --> CLAUDE
    ROUTER --> OLLAMA
```

---

## 3. Triage Akışı (False Positive Filtresi)

```mermaid
sequenceDiagram
    participant S as Static Engine
    participant AI as AI Triage
    participant DB as Database
    participant UI as Web UI

    S->>DB: 47 issue kaydet
    DB->>AI: Yeni issue'ları gönder
    
    loop Her CRITICAL+ issue
        AI->>AI: Cache kontrol
        alt Cache miss
            AI->>AI: LLM: "Bu gerçek mi?"
            AI->>AI: confidence score hesapla
            AI->>DB: ai.triage kaydet
        end
    end
    
    AI->>DB: priority_score güncelle
    DB->>UI: 35 gerçek + 12 muhtemel FP göster
    
    Note over UI: FP olanlar soluk gösterilir<br/>"Muhtemelen false positive" badge
```

---

## 4. Semantic Security Scan

```mermaid
flowchart LR
    A[Değişen dosyalar] --> B[Context Builder]
    B --> C[Code Snippet]
    B --> D[Imports / Callers]
    B --> E[Data Flow hints]
    
    C --> F[PII Redactor]
    D --> F
    E --> F
    
    F --> G[LLM Prompt]
    G --> H{Structured JSON response}
    H --> I[Yeni AI issue'lar]
    H --> J[Mevcut issue doğrulama]
    
    I --> K[Fingerprint + Dedup]
    J --> K
    K --> L[Issue DB'ye merge]
```

---

## 5. RAG Codebase Chat

```mermaid
graph TD
    subgraph "Indexing (bir kez / per analysis)"
        CODE[Source Code] --> CHUNK[Chunker]
        CHUNK --> EMB1[Embedding Model]
        EMB1 --> VDB[(Vector DB)]
        
        ISSUES[Issue History] --> EMB1
        METRICS[Metrics] --> VDB
    end
    
    subgraph "Query"
        Q[User Question] --> EMB2[Embedding Model]
        EMB2 --> SEARCH[Similarity Search]
        VDB --> SEARCH
        SEARCH --> TOPK[Top-K Chunks]
        TOPK --> LLM[LLM Answer]
        LLM --> ANS[Response + Sources]
    end
```

---

## 6. AI Faz Timeline

```mermaid
gantt
    title AI Özellikleri Timeline
    dateFormat YYYY-MM-DD
    
    section Statik Temel
    Faz 1 CLI Scanner       :s1, 2026-07-08, 21d
    Faz 2 Server + API        :s2, after s1, 28d
    
    section AI Katmanı
    AI-1 Issue Explain        :ai1, after s2, 7d
    AI-2 FP Triage            :ai2, after ai1, 14d
    AI-3 Semantic Security    :ai3, after ai2, 21d
    AI-4 Auto-Fix + PR Summary :ai4, after ai3, 21d
    AI-5 NL Rules             :ai5, after ai4, 14d
    AI-6 RAG Chat             :ai6, after ai5, 28d
```

---

## 7. Maliyet Karar Ağacı

```mermaid
flowchart TD
    A[Issue oluştu] --> B{Severity?}
    B -->|INFO/MINOR| C[❌ AI çağırma]
    B -->|MAJOR| D{User requested?}
    B -->|CRITICAL+| E[✅ Full AI pipeline]
    
    D -->|Evet| F[✅ Explain only]
    D -->|Hayır| C
    
    E --> G{Token budget OK?}
    G -->|Hayır| H[⚠️ Queue / skip]
    G -->|Evet| I{Cache hit?}
    I -->|Evet| J[Cache'den dön]
    I -->|Hayır| K[LLM API call]
    
    K --> L{Provider?}
    L -->|Ollama| M[Free, local]
    L -->|OpenAI/Claude| N[Token cost]
```
