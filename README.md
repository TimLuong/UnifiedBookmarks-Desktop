# 🔖 UnifiedBookmarks Desktop

A desktop app to **collect, deduplicate, AI-categorize, and sync** bookmarks across all your Chrome / Edge browser profiles — in one place.

Built with [Wails v2](https://wails.io) (Go backend + Svelte frontend). Runs natively on Windows.

---

## ✨ What It Does

| Step | What happens |
|------|-------------|
| **1 · Scan** | Detects all Chrome/Edge profiles installed on your machine |
| **2 · Collect** | Reads every bookmark from every profile, deduplicates across profiles |
| **3 · Analyze** | Sends bookmarks to an AI (OpenAI / Azure OpenAI / local LLM) which adds PARA categories and tags |
| **4 · Sync** | Writes the enriched bookmarks back into every browser profile's Bookmarks file |

Everything runs **locally** — your bookmarks never leave your machine except for the AI categorization call (which you can also point at a local LLM like Ollama or LM Studio).

---

## 🖥 Screenshot

The sidebar shows your pipeline progress, Collections (browser profiles), PARA categories, and Tags.  
The main panel lists bookmarks with their AI-assigned category, tags, relevance score, and folder path.

---

## 🚀 Installation (pre-built EXE)

1. Download `UnifiedBookmarks-Desktop.exe` from [Releases](../../releases)
2. Place it in any folder — it is **fully portable** (no installer needed)
3. In the same folder, create a `.env` file (copy from `.env.example`):

```env
OPENAI_API_KEY=your-api-key-here
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o
```

4. Double-click the EXE and run the pipeline: **Scan → Collect → Analyze → Sync**

> The `.env` file lives next to the EXE and is **never uploaded anywhere**.

---

## 🔨 Build It Yourself (from source)

Building from source lets you verify exactly what code runs on your machine.

### Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | ≥ 1.23 | https://go.dev/dl |
| Node.js | ≥ 18 | https://nodejs.org |
| Wails CLI | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |

### Steps

```bash
# 1. Clone the repo
git clone https://github.com/your-user/UnifiedBookmarks-Desktop
cd UnifiedBookmarks-Desktop

# 2. Install frontend deps
cd frontend && npm install && cd ..

# 3. Build the EXE
wails build
```

The output EXE is at:
```
build/bin/UnifiedBookmarks-Desktop.exe
```

Copy it wherever you want along with your `.env` file.

### Live development mode

```bash
wails dev
```

Opens the app with hot-reload. A dev server also runs at `http://localhost:34115` for browser devtools access.

---

## 🤖 AI / LLM Configuration

The app reads settings from a `.env` file placed **next to the EXE**:

```env
# Required
OPENAI_API_KEY=your-key

# Defaults shown — change as needed
OPENAI_BASE_URL=https://api.openai.com/v1   # Azure: https://your-resource.openai.azure.com/openai/v1/
OPENAI_MODEL=gpt-4o                          # or gpt-4o-mini, gpt-5.4, etc.
BATCH_SIZE=0                                 # 0 = send all at once; set to e.g. 50 for large collections
MAX_RETRIES=3
CONFIDENCE_THRESHOLD=0.7
```

### Local LLM (Ollama / LM Studio)

```env
OPENAI_BASE_URL=http://localhost:11434/v1    # Ollama
OPENAI_API_KEY=ollama                        # any non-empty string
OPENAI_MODEL=llama3
```

### Custom AI prompt

Place a `ai-prompt.txt` file next to the EXE to override the system prompt used for categorization.

---

## 🛡 Security & Malware Scanning

This app is **open source** — you can read every line of Go and Svelte code before running it.

If you downloaded a pre-built EXE from an untrusted source, or want to verify the release EXE is clean:

### Option 1 — VirusTotal (easiest)

1. Go to https://www.virustotal.com
2. Click **Choose file** → select `UnifiedBookmarks-Desktop.exe`
3. Wait for 70+ antivirus engines to scan it
4. Any result with 0–2 positives on a Wails/Go app is normal (Go binaries sometimes trigger heuristic false-positives)

### Option 2 — Windows Defender offline scan

```powershell
# Run in PowerShell as Administrator
Start-MpScan -ScanType CustomScan -ScanPath "C:\path\to\UnifiedBookmarks-Desktop.exe"
```

### Option 3 — Build from source (most trustworthy)

Follow the **Build It Yourself** section above. You compile the code yourself — there is no binary trust required.

### What this app does NOT do

- ❌ It does NOT connect to any remote server except your configured AI endpoint
- ❌ It does NOT send your bookmarks anywhere except the AI call you configure
- ❌ It does NOT install services, registry entries, or background processes
- ❌ It does NOT require admin privileges
- ✅ All bookmark data stays locally in `build/bin/backups/`

You can monitor network activity with [Wireshark](https://www.wireshark.org/) or [glasswire](https://www.glasswire.com/) to verify.

---

## 📁 File Layout

```
UnifiedBookmarks-Desktop.exe   ← the app
.env                            ← your API key (never commit this)
.env.example                    ← template for .env
ai-prompt.txt                   ← optional: custom AI system prompt
backups/                        ← automatic backups before every sync
cache/                          ← AI analysis cache (avoids re-calling AI)
```

---

## 🔒 Privacy Notes

- **Bookmark data** is read directly from your local Chrome/Edge profile files. It never leaves your machine except in the AI categorization call.
- **Backups** of your original bookmark files are saved locally in `backups/` before every sync.
- **`.env`** contains your API key — keep it private, never commit it to git.
- The app does not create any account, does not phone home, and has no telemetry.

---

## 🧰 Tech Stack

- **Backend**: Go 1.23, [Wails v2](https://wails.io)
- **Frontend**: Svelte + TypeScript + Vite
- **AI**: OpenAI-compatible API (works with OpenAI, Azure OpenAI, local LLMs)
- **Platform**: Windows (Chrome/Edge bookmark paths are Windows-specific)

---

## 📄 License

MIT
