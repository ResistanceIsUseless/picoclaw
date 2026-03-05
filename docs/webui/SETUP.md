# CLAW Web UI Setup Guide

Complete setup instructions for the CLAW Web UI.

## Prerequisites

- **Node.js 18+** and npm
- **Go 1.21+** (for backend)
- **CLAW backend** with web UI support

## Quick Start

### 1. Install Frontend Dependencies

```bash
cd web
npm install
```

This will install:
- React 18 + React DOM
- TypeScript
- Vite (build tool)
- TailwindCSS
- React Router
- React Query
- D3.js

### 2. Verify Installation

```bash
npm run build
```

Should complete without errors and create a `dist/` directory.

### 3. Start Development Server

```bash
npm run dev
```

Output should show:
```
VITE v5.1.6  ready in 500 ms

➜  Local:   http://localhost:5173/
➜  Network: use --host to expose
```

### 4. Start CLAW Backend

In a separate terminal:

```bash
cd ..  # Back to project root
./build/test-claw -target example.com -webui :8080 -dry-run
```

The `-dry-run` flag allows testing without an API key.

### 5. Open UI

Navigate to http://localhost:5173 in your browser.

## Troubleshooting

### Port Already in Use

If port 5173 is taken:

```bash
npm run dev -- --port 5174
```

Update proxy in `vite.config.ts` if backend isn't on 8080.

### WebSocket Connection Failed

Check that:
1. Backend is running with `-webui :8080`
2. Browser console shows connection attempts
3. Firewall allows WebSocket connections

### API 404 Errors

Verify backend is running:

```bash
curl http://localhost:8080/api/status
```

Should return JSON with system status.

### Build Errors

Clear node_modules and reinstall:

```bash
rm -rf node_modules package-lock.json
npm install
```

## Development Workflow

### Hot Module Replacement

Vite provides instant updates when you edit files:

1. Edit `src/components/Pipeline/PipelineView.tsx`
2. Save
3. Browser updates automatically

### API Proxy

Vite proxies API requests to backend:

```
http://localhost:5173/api/status
  ↓
http://localhost:8080/api/status
```

This avoids CORS issues during development.

### TypeScript Type Checking

```bash
npm run build
```

Runs `tsc` to check for type errors.

## Production Build

### Build for Production

```bash
npm run build
```

Creates optimized bundle in `dist/`:
```
dist/
├── index.html
├── assets/
│   ├── index-[hash].js
│   └── index-[hash].css
└── vite.svg
```

### Preview Production Build

```bash
npm run preview
```

Serves the production build locally for testing.

### Embed in Go Binary

To serve the frontend from the Go backend:

1. Build the frontend: `npm run build`
2. Embed `dist/` in Go using `embed` package
3. Serve static files from backend

Example in `pkg/webui/server.go`:

```go
//go:embed dist/*
var distFS embed.FS

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
    http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
}
```

## Testing

### Manual Testing Checklist

- [ ] Pipeline view loads
- [ ] WebSocket connects (green indicator)
- [ ] Phase card shows current phase
- [ ] Tool execution log updates in real-time
- [ ] Tools view lists all tools
- [ ] Navigation between views works
- [ ] Progress bars update
- [ ] API calls succeed (check Network tab)

### Testing with Live Backend

```bash
# Terminal 1: Start backend with API key
export OPENROUTER_API_KEY=your-key
./build/test-claw -target example.com -webui :8080 -provider openrouter

# Terminal 2: Start frontend
cd web && npm run dev
```

Watch the UI update in real-time as CLAW executes.

## Next Steps

Once the basic UI is working:

1. **Implement Graph Visualization** (Phase 5)
   - D3.js force-directed layout
   - Frontier highlighting
   - Interactive node details

2. **Add Artifact Browser**
   - Filterable artifact list
   - JSON viewer
   - Export functionality

3. **Enhance Pipeline View**
   - Execution timeline
   - Detailed tool output
   - Contract validation details

4. **Add Admin Features**
   - Pipeline configuration editor
   - Model routing settings
   - Tool tier management

## Resources

- [Vite Documentation](https://vitejs.dev/)
- [React Query](https://tanstack.com/query/latest)
- [TailwindCSS](https://tailwindcss.com/)
- [D3.js](https://d3js.org/)
