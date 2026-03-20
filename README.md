<p align="center">
  <img src="assets/logo.png" alt="Helios" width="300">
</p>

<h1 align="center">Helios</h1>

<p align="center">A fast, native PostgreSQL IDE built with Go and Fyne.</p>

---

## Features

- **SQL Editor** with context-aware autocomplete (tables, columns, aliases, dot notation)
- **Schema Browser** with lazy-loaded tree: schemas, tables, columns, indexes, constraints, triggers, functions
- **Query Results** with alternating row stripes, bold headers, and streaming for large result sets (50k row cap)
- **EXPLAIN ANALYZE** visualization via embedded pev2 — opens in browser with one click or Ctrl+E
- **Export** results to CSV, TSV, or XLSX
- **Transaction Management** — Begin, Commit, Rollback with visual indicators
- **Query History** — persistent, searchable, click to re-use
- **File Editor** — browse directories, edit files, virtualized viewer for large files
- **Function Viewer** — click any function to see its full `CREATE OR REPLACE` definition, with "Open in Terminal" to edit
- **Import Connections** from JSON (single object or array)
- **Multiple Tabs** — terminals and file editors side by side
- **Dark/Light Theme** — M3-inspired color system with JetBrains Mono

## Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| Ctrl+Enter | Run query |
| Ctrl+E | EXPLAIN ANALYZE |
| Ctrl+T | New terminal |
| Ctrl+W | Close tab |
| Ctrl+S | Save file |

## Build

### Prerequisites (Ubuntu/Debian)

```bash
sudo apt install -y gcc libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
```

### Prerequisites (Fedora/RHEL)

```bash
sudo dnf install -y gcc mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

### Prerequisites (macOS)

Xcode command line tools only — no extra packages needed:

```bash
xcode-select --install
```

### Compile and run

```bash
go build -o helios .
./helios
```

Requires Go 1.24+.

## License

MIT
