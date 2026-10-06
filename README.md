# Agent Pribadi - Unified MCP & Multi-Agent Engine

Library Go untuk orkestrasi agent otonom dengan Model Context Protocol (MCP) terpadu.

---

## Fitur Utama

- **Unified In-Memory MCP Registry**: Built-in tools dan Custom tools milik pengguna berada dalam satu wadah registry `mcp.Server`.
- **Zero Schema Duplication**: Skema LLM (OpenAI-compatible) otomatis di-generate secara dinamis dari definisi tool MCP.
- **Single-Line Dispatch**: Eksekusi tool langsung didelegasikan ke `server.Call(ctx, name, args)` dengan proteksi timeout, panic recovery, dan permission checks.
- **Two-Tier Orchestration**: 
  - `TaskMaker`: Memecah query pengguna menjadi tahapan kerja (*stages*).
  - `Orchestrator`: Mengatur eksekusi dan mendelegasikan tugas ke `subAgent`.
  - `subAgent`: Worker ReAct loop yang menjalankan tools untuk menyelesaikan instruksi.

---

## Struktur Direktori

```
agent_pribadi/
├── agent/                  # Core Agent Engine
│   ├── chatGenerate.go    # HTTP client ke LLM API (OpenRouter/OpenAI)
│   ├── orchestrator.go    # TaskMaker & Orchestrator workflow
│   ├── sub-agent.go       # Worker sub-agent ReAct loop
│   ├── toolcall.go        # MCP-to-LLM schema converter & executor
│   └── toolcall_test.go   # Unit test integrasi MCP
├── controller/             # Public Types & Setup
│   └── type_setUp.go      # SubAgent config, RequestChat, options
└── mcp/                    # Unified MCP Layer
    ├── builtin.go         # 7 built-in tools & DefaultServer()
    ├── permission.go      # Permission interface & policies
    ├── schema.go          # Automatic JSON Schema generator
    ├── server.go          # In-memory MCP Server & Stdio runner
    ├── tool.go            # Type-safe Tool builder (mcp.Func)
    ├── client/            # Logging & credentials helper
    └── tools/             # Low-level system implementations
```

---

## Panduan Penggunaan

### 1. Menjalankan dengan Built-in Tools (Zero Config)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "agentPribadi/agent"
)

func main() {
    ctx := context.Background()

    // Cukup panggil Orchestrator, otomatis memuat mcp.DefaultServer()
    result, err := agent.Orchestrator(
        ctx,
        "Analisis struktur kode di direktori ini dan buat rangkuman di summary.txt",
        nil, "", nil,
    )
    if err != nil {
        log.Fatalf("Error: %v", err)
    }

    fmt.Println(result)
}
```

---

### 2. Menambahkan Custom MCP Tool

```go
package main

import (
    "context"
    "fmt"
    "log"

    "agentPribadi/agent"
    "agentPribadi/mcp"
)

type StockInput struct {
    ProductID string `json:"product_id" desc:"ID produk yang dicari"`
}

func main() {
    ctx := context.Background()

    // Inisialisasi server default
    server := mcp.DefaultServer()

    // Tambah custom tool
    server.Add(mcp.Func(
        "check_stock",
        "Periksa stok produk di database",
        func(ctx context.Context, in StockInput) (any, error) {
            return map[string]any{"product_id": in.ProductID, "stock": 42}, nil
        },
        mcp.ReadOnly(),
    ))

    // Jalankan dengan server terpadu
    result, err := agent.Orchestrator(
        ctx,
        "Cek stok untuk produk PROD-01 dan simpan laporannya",
        nil, "", server,
    )
    if err != nil {
        log.Fatalf("Error: %v", err)
    }

    fmt.Println(result)
}
```

---

## Built-in Tools

| Tool | Fungsi | Parameter Utama |
| :--- | :--- | :--- |
| `bash` | Menjalankan perintah terminal | `command` |
| `read` | Membaca file teks dengan paging | `file_path`, `offset`, `limit` |
| `write` | Membuat file baru | `file_path`, `content` |
| `edit` | Mengubah teks dalam file secara atomik | `file_path`, `old_content`, `new_content` |
| `multi_edit` | Mengubah beberapa file sekaligus | `edits` |
| `glob` | Mencari file berdasarkan pattern | `pattern`, `path` |
| `grep` | Mencari teks regex di direktori | `pattern`, `path`, `include` |

---

## Menjalankan Test

```powershell
go test ./... -v
```
