# CLAW Web UI

Modern React + TypeScript web interface for CLAW security assessment framework.

## Features

- **Pipeline View**: Real-time monitoring of phase execution with contract tracking
- **Graph View**: Knowledge graph visualization (D3.js force-directed layout)
- **Tools View**: Browse available security tools by tier
- **WebSocket Updates**: Live event streaming from backend
- **Responsive Design**: TailwindCSS with dark theme

## Development

### Prerequisites

- Node.js 18+ and npm
- CLAW backend running on port 8080

### Install Dependencies

```bash
cd web
npm install
```

### Start Development Server

```bash
npm run dev
```

This starts Vite dev server on http://localhost:5173 with proxy to backend API on :8080.

### Build for Production

```bash
npm run build
```

Output goes to `dist/` directory.

## Architecture

### Tech Stack

- **React 18** - UI framework
- **TypeScript** - Type safety
- **Vite** - Fast build tool
- **TailwindCSS** - Styling
- **React Query** - API state management
- **React Router** - Client-side routing
- **D3.js** - Graph visualization
- **WebSocket** - Real-time updates

### Project Structure

```
web/
├── src/
│   ├── api/
│   │   └── client.ts          # API client for backend
│   ├── hooks/
│   │   └── useWebSocket.ts    # WebSocket hook
│   ├── components/
│   │   ├── Pipeline/
│   │   │   ├── PipelineView.tsx       # Main dashboard
│   │   │   ├── PhaseCard.tsx          # Phase progress card
│   │   │   └── ToolExecutionLog.tsx   # Tool activity log
│   │   ├── Graph/
│   │   │   └── GraphView.tsx          # Knowledge graph (placeholder)
│   │   └── Tools/
│   │       └── ToolsView.tsx          # Tool registry browser
│   ├── App.tsx               # Main app with routing
│   ├── main.tsx              # Entry point
│   └── index.css             # Global styles
├── package.json
├── vite.config.ts
├── tailwind.config.js
└── tsconfig.json
```

## Usage with CLAW

### Start Backend with Web UI

```bash
./build/test-claw -target example.com -webui :8080 -provider openrouter
```

### Start Frontend Dev Server

```bash
cd web
npm run dev
```

### Access UI

Open http://localhost:5173 in your browser.

The frontend proxies API requests to the backend on :8080.

## WebSocket Events

The UI subscribes to WebSocket events for real-time updates:

- `phase_start` - Phase execution started
- `phase_complete` - Phase execution completed
- `tool_execution` - Tool execution status update
- `artifact` - New artifact published
- `graph_update` - Knowledge graph updated
- `log` - Log message

## API Endpoints

All endpoints are prefixed with `/api`:

- `GET /api/status` - System health
- `GET /api/pipeline/status` - Current pipeline status
- `GET /api/phase` - Current phase details
- `GET /api/artifacts` - Query artifacts
- `GET /api/graph/nodes` - Graph nodes
- `GET /api/graph/edges` - Graph edges
- `GET /api/tools` - Tool registry

## Next Steps

### Phase 5: Graph Visualization

Implement D3.js force-directed graph:

1. Create D3 force simulation
2. Render nodes (color by type)
3. Highlight frontier nodes
4. Add zoom/pan controls
5. Node click for details
6. Real-time updates via WebSocket

### Phase 6: Enhanced Features

- Artifact browser with filtering
- Export reports (PDF/HTML)
- Custom queries on graph
- Pipeline configuration UI
- Model routing settings
