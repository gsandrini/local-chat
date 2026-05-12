# Development

## Project structure

```txt
/
├── app.go                         # Core application logic
├── main.go                        # Entry point
│
├── frontend/                      # UI layer
│   ├── index.html
│   ├── package.json
│   ├── tailwind.config.js
│   │
│   └── assets/
│       ├── css/
│       │   ├── tailwind.src.css  # Edit this file
│       │   └── tailwind.min.css  # Generated (compiled)
│       │
│       └── js/
│           ├── app.js            # Frontend app logic
│           └── alpine.min.js     # Local Alpine bundle
│
├── docs/                          # Technical documentation
│
├── .github/
│   └── workflows/
│       └── release.yml           # CI/CD pipeline
│
├── go.mod
└── wails.json
```

---

## Architecture

```txt
Frontend (Alpine.js)
    │
    │ invoke(params)
    ▼
Backend (Go / Wails)
    │
    │ business logic
    ▼
System / External Services
```

`app.go` contains the core application logic exposed to the frontend via Wails bindings.

---

## Prerequisites

- [Go](https://golang.org/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+ (only needed to rebuild Tailwind CSS)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/)

### Go

```bash
# Install Go
sudo snap install go --classic
source ~/.bashrc

# Check
go version
```

### Wails

```bash
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Add Go bin to the PATH
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc

# Check
wails version
```

### Linux

```bash
sudo apt install libwebkit2gtk-4.1-dev build-essential libssl-dev \
  libgtk-3-dev librsvg2-dev
```

---

## Run in development

```bash
# Install Go dependencies
go mod tidy

# Run wails
wails dev -tags webkit2_41
```

Wails automatically runs `npm install` and starts the Tailwind watcher
(`npm run watch`) as a background process, as configured in `wails.json`:

```txt
"frontend:install": "npm install",
"frontend:build": "npm run build",
"frontend:dev:watcher": "npm run watch"
```

Hot reload is active: changes to `frontend/` are reflected instantly
without restarting. Changes to `app.go` require a recompile (Wails
handles this automatically).

---

## Rebuild Tailwind CSS

Only needed if you want to manage Tailwind manually, outside of `wails dev`.

```bash
cd frontend
npm install

# One-shot build (minified output)
npm run build

# Watch mode (rebuilds on every file change)
npm run watch
```

---

## Build

```bash
wails build
```

Output is placed in `build/bin/`.

The binary is fully self-contained. On Linux `libwebkit2gtk` must be available on the target system.
