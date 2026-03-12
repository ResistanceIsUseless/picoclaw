package webui

import "fmt"

func renderDashboardHTML() string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>PicoClaw Live Dashboard</title>
  <style>
    :root {
      --bg: #f4f1ea;
      --panel: rgba(255,255,255,0.72);
      --panel-strong: rgba(255,255,255,0.9);
      --line: rgba(44, 52, 64, 0.12);
      --text: #1f2933;
      --muted: #66788a;
      --accent: #0f766e;
      --accent-2: #c2410c;
      --ok: #166534;
      --warn: #b45309;
      --danger: #b91c1c;
      --shadow: 0 22px 70px rgba(34, 41, 47, 0.12);
      --radius: 22px;
      --font-sans: "Avenir Next", "Segoe UI", sans-serif;
      --font-mono: "JetBrains Mono", "SFMono-Regular", monospace;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: var(--font-sans);
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(15,118,110,0.14), transparent 30%%),
        radial-gradient(circle at top right, rgba(194,65,12,0.16), transparent 28%%),
        linear-gradient(180deg, #f7f5ef 0%%, #ece7dc 100%%);
      min-height: 100vh;
    }
    .shell {
      max-width: 1400px;
      margin: 0 auto;
      padding: 28px 22px 40px;
    }
    .hero {
      display: flex;
      gap: 20px;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 22px;
    }
    .hero-card, .meta-card, .panel {
      background: var(--panel);
      backdrop-filter: blur(18px);
      border: 1px solid var(--line);
      box-shadow: var(--shadow);
      border-radius: var(--radius);
    }
    .hero-card {
      flex: 1;
      padding: 28px;
      min-height: 170px;
    }
    .meta-card {
      width: 320px;
      padding: 24px;
    }
    .eyebrow {
      text-transform: uppercase;
      letter-spacing: 0.16em;
      font-size: 12px;
      color: var(--accent);
      margin-bottom: 12px;
      font-weight: 700;
    }
    h1 {
      margin: 0 0 8px;
      font-size: clamp(28px, 4vw, 46px);
      line-height: 1.05;
    }
    .sub {
      color: var(--muted);
      max-width: 48rem;
      font-size: 15px;
      line-height: 1.6;
    }
    .hero-badges {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 18px;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 10px 14px;
      border-radius: 999px;
      background: rgba(15,118,110,0.08);
      color: var(--accent);
      font-size: 13px;
      font-weight: 700;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, 1fr);
      gap: 18px;
    }
    .panel { padding: 20px; }
    .panel h2 {
      margin: 0 0 14px;
      font-size: 18px;
    }
    .kpi-grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 14px;
      margin-bottom: 18px;
    }
    .kpi {
      background: var(--panel-strong);
      border-radius: 18px;
      padding: 16px;
      border: 1px solid var(--line);
    }
    .kpi .label {
      color: var(--muted);
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      margin-bottom: 8px;
    }
    .kpi .value {
      font-size: 28px;
      font-weight: 800;
    }
    .stack { display: flex; flex-direction: column; gap: 14px; }
    .phase-card, .artifact-card, .log-entry, .tool-card, .graph-card {
      background: var(--panel-strong);
      border-radius: 18px;
      border: 1px solid var(--line);
      padding: 16px;
    }
    .tool-row, .meta-row {
      display: flex;
      gap: 12px;
      justify-content: space-between;
      align-items: center;
      font-size: 13px;
      color: var(--muted);
      margin-top: 8px;
    }
    .metric-list {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 10px;
      margin-top: 14px;
    }
    .metric {
      padding: 12px;
      border-radius: 14px;
      background: rgba(15,118,110,0.06);
      border: 1px solid rgba(15,118,110,0.08);
    }
    .metric .value {
      font-size: 22px;
      font-weight: 800;
      color: var(--accent);
    }
    .metric .label {
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--muted);
      margin-top: 4px;
    }
    .tool-list {
      display: flex;
      flex-direction: column;
      gap: 10px;
      margin-top: 14px;
    }
    .tool-pill {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      align-items: center;
      padding: 12px 14px;
      border-radius: 14px;
      background: rgba(31,41,51,0.04);
    }
    .event-type {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      border-radius: 999px;
      padding: 5px 10px;
      background: rgba(15,118,110,0.08);
      color: var(--accent);
      font-size: 12px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.06em;
    }
    .event-type.log { background: rgba(100,116,139,0.12); color: #475569; }
    .event-type.graph_update { background: rgba(2,132,199,0.12); color: #0369a1; }
    .event-type.artifact { background: rgba(194,65,12,0.12); color: var(--accent-2); }
    .event-type.phase_complete { background: rgba(22,101,52,0.12); color: var(--ok); }
    .tab-row {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin-bottom: 14px;
    }
    .tab-button {
      border: 0;
      border-radius: 999px;
      padding: 8px 12px;
      background: rgba(31,41,51,0.06);
      color: var(--muted);
      font: inherit;
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.04em;
      cursor: pointer;
    }
    .tab-button.active {
      background: rgba(15,118,110,0.12);
      color: var(--accent);
    }
    .summary-list {
      display: flex;
      flex-direction: column;
      gap: 10px;
    }
    .node-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
    }
    .search-row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-bottom: 14px;
    }
    .search-input, .filter-select {
      border: 1px solid var(--line);
      border-radius: 14px;
      background: rgba(255,255,255,0.78);
      color: var(--text);
      font: inherit;
      padding: 10px 12px;
    }
    .search-input {
      flex: 1;
      min-width: 220px;
    }
    .filter-select {
      min-width: 180px;
    }
    .node-card {
      padding: 14px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: rgba(31,41,51,0.04);
      cursor: pointer;
      transition: transform 160ms ease, border-color 160ms ease, background 160ms ease;
    }
    .node-card:hover {
      transform: translateY(-1px);
      border-color: rgba(15,118,110,0.22);
      background: rgba(15,118,110,0.06);
    }
    .node-card.active {
      border-color: rgba(15,118,110,0.38);
      background: rgba(15,118,110,0.1);
    }
    .node-header {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      align-items: center;
      margin-bottom: 8px;
    }
    .detail-card {
      padding: 16px;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: var(--panel-strong);
    }
    .property-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      margin-top: 12px;
    }
    .property-row {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: flex-start;
      padding: 10px 12px;
      border-radius: 12px;
      background: rgba(31,41,51,0.04);
    }
    .summary-row {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: center;
      padding: 12px 14px;
      border-radius: 14px;
      background: rgba(31,41,51,0.04);
    }
    .phase-card header, .artifact-card header, .tool-card header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      margin-bottom: 10px;
    }
    .phase-status, .pill {
      border-radius: 999px;
      padding: 6px 10px;
      font-size: 12px;
      font-weight: 700;
      background: rgba(15,118,110,0.1);
      color: var(--accent);
    }
    .phase-status.warn { color: var(--warn); background: rgba(180,83,9,0.12); }
    .phase-status.danger { color: var(--danger); background: rgba(185,28,28,0.12); }
    .muted { color: var(--muted); }
    .mono { font-family: var(--font-mono); }
    pre {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      font-family: var(--font-mono);
      font-size: 12px;
      line-height: 1.5;
      color: #24323f;
    }
    .span-8 { grid-column: span 8; }
    .span-4 { grid-column: span 4; }
    .span-6 { grid-column: span 6; }
    .span-12 { grid-column: span 12; }
    .progress {
      height: 12px;
      border-radius: 999px;
      overflow: hidden;
      background: rgba(31,41,51,0.08);
    }
    .progress > div {
      height: 100%%;
      background: linear-gradient(90deg, var(--accent), #14b8a6);
      width: 0%%;
      transition: width 280ms ease;
    }
    .empty {
      color: var(--muted);
      text-align: center;
      padding: 24px;
    }
    @media (max-width: 1000px) {
      .hero { flex-direction: column; }
      .meta-card { width: 100%%; }
      .kpi-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .span-8, .span-4, .span-6 { grid-column: span 12; }
    }
    @media (max-width: 640px) {
      .shell { padding: 18px 14px 28px; }
      .hero-card, .meta-card, .panel { padding: 18px; }
      .kpi-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="hero-card">
        <div class="eyebrow">Local Mission Control</div>
        <h1>PicoClaw Live Dashboard</h1>
        <div class="sub">A local-first view into pipeline execution, graph growth, artifacts, tools, and live events. Designed for same-machine usage with zero extra deployment steps.</div>
        <div class="hero-badges">
          <div class="badge">Realtime pipeline state</div>
          <div class="badge">Artifact feed</div>
          <div class="badge">Graph frontier</div>
          <div class="badge">Tool inventory</div>
        </div>
      </div>
      <aside class="meta-card">
        <div class="eyebrow">Connection</div>
        <div id="connectionState" class="pill">Connecting</div>
        <div style="margin-top:14px" class="muted">This dashboard is served from the running PicoClaw process on your local machine.</div>
      </aside>
    </section>

    <section class="kpi-grid">
      <div class="kpi"><div class="label">Pipeline</div><div id="pipelineName" class="value">-</div></div>
      <div class="kpi"><div class="label">Status</div><div id="pipelineStatus" class="value">-</div></div>
      <div class="kpi"><div class="label">Artifacts</div><div id="artifactCount" class="value">0</div></div>
      <div class="kpi"><div class="label">Graph Nodes</div><div id="graphNodes" class="value">0</div></div>
    </section>

    <section class="grid">
      <div class="panel span-8">
        <h2>Pipeline Progress</h2>
        <div class="progress"><div id="progressBar"></div></div>
        <div id="phaseSummary" class="stack" style="margin-top:16px"></div>
      </div>
      <div class="panel span-4">
        <h2>Current Phase</h2>
        <div id="currentPhase" class="stack"><div class="empty">No active phase yet.</div></div>
      </div>
      <div class="panel span-6">
        <h2>Recent Artifacts</h2>
        <div id="artifacts" class="stack"><div class="empty">Artifacts will appear here.</div></div>
      </div>
      <div class="panel span-6">
        <h2>Tool Catalog</h2>
        <div id="tools" class="stack"><div class="empty">Loading tools...</div></div>
      </div>
      <div class="panel span-6">
        <h2>Graph Frontier</h2>
        <div id="frontier" class="graph-card muted">Loading graph frontier...</div>
      </div>
      <div class="panel span-6">
        <h2>Graph Explorer</h2>
        <div class="search-row">
          <input id="graphSearch" class="search-input" type="search" placeholder="Search nodes by label or ID">
          <select id="graphTypeFilter" class="filter-select"></select>
        </div>
        <div id="graphExplorer" class="stack"><div class="empty">Loading graph nodes...</div></div>
      </div>
      <div class="panel span-6">
        <h2>Node Details</h2>
        <div id="nodeDetails" class="detail-card muted">Select a node from the graph explorer to inspect its properties and edges.</div>
      </div>
      <div class="panel span-6">
        <h2>Live Events</h2>
        <div id="eventFilters" class="tab-row"></div>
        <div id="events" class="stack"><div class="empty">Waiting for live events...</div></div>
      </div>
    </section>
  </div>

  <script>
    const state = { events: [], graphNodes: [], graphEdges: [], selectedNodeId: null, eventFilter: 'all', graphSearch: '', graphType: 'all' };
    const el = (id) => document.getElementById(id);
    const eventFilters = ['all', 'phase_start', 'phase_complete', 'tool_execution', 'artifact', 'graph_update', 'log'];

    function badgeClass(status) {
      const value = String(status || '').toLowerCase();
      if (value.includes('fail') || value.includes('error')) return 'phase-status danger';
      if (value.includes('block') || value.includes('warn')) return 'phase-status warn';
      return 'phase-status';
    }

    function renderTools(items) {
      const root = el('tools');
      if (!items || items.length === 0) {
        root.innerHTML = '<div class="empty">No tools loaded.</div>';
        return;
      }
      root.innerHTML = items.slice(0, 12).map(tool =>
        '<div class="tool-card">' +
          '<header>' +
            '<strong>' + tool.name + '</strong>' +
            '<span class="pill mono">tier ' + tool.tier + '</span>' +
          '</header>' +
          '<div class="muted">' + (tool.description || 'No description available.') + '</div>' +
          '<div class="tool-row"><span>Ready in runtime</span><span class="mono">requestable</span></div>' +
        '</div>'
      ).join('');
    }

    function renderArtifacts(items) {
      const root = el('artifacts');
      if (!items || items.length === 0) {
        root.innerHTML = '<div class="empty">Artifacts will appear here.</div>';
        return;
      }
      root.innerHTML = items.slice(0, 8).map(item =>
        '<article class="artifact-card">' +
          '<header>' +
            '<strong>' + item.type + '</strong>' +
            '<span class="pill">' + (item.phase || 'unknown phase') + '</span>' +
          '</header>' +
          '<div class="muted">Domain: ' + (item.domain || 'unknown') + ' · ' + new Date(item.created_at).toLocaleTimeString() + '</div>' +
          '<pre>' + JSON.stringify(item.data, null, 2) + '</pre>' +
        '</article>'
      ).join('');
    }

    function renderCurrentPhase(detail) {
      const root = el('currentPhase');
      if (!detail) {
        root.innerHTML = '<div class="empty">No active phase yet.</div>';
        return;
      }
      const tools = (detail.dag_state && detail.dag_state.tools) || [];
      const completed = tools.filter(tool => tool.status === 'COMPLETED').length;
      const running = tools.filter(tool => tool.status === 'RUNNING').length;
      const blocked = tools.filter(tool => tool.status === 'BLOCKED').length;
      const contract = detail.contract || {};
      const requiredTools = contract.required_tools || [];
      const requiredProfiles = contract.required_profiles || [];
      const requiredArtifacts = contract.required_artifacts || [];
      const toolRows = tools.slice(0, 6).map(tool =>
        '<div class="tool-pill">' +
          '<div><strong>' + tool.name + '</strong><div class="muted">' + (tool.summary || 'No summary yet.') + '</div></div>' +
          '<span class="' + badgeClass(tool.status) + '">' + tool.status + '</span>' +
        '</div>'
      ).join('');
      root.innerHTML =
        '<div class="phase-card">' +
          '<header>' +
            '<strong>' + detail.name + '</strong>' +
            '<span class="' + badgeClass(detail.status) + '">' + detail.status + '</span>' +
          '</header>' +
          '<div class="muted">Iteration ' + detail.iteration + ' / ' + detail.max_iterations + '</div>' +
          '<div class="metric-list">' +
            '<div class="metric"><div class="value">' + completed + '</div><div class="label">completed tools</div></div>' +
            '<div class="metric"><div class="value">' + running + '</div><div class="label">running</div></div>' +
            '<div class="metric"><div class="value">' + blocked + '</div><div class="label">blocked</div></div>' +
            '<div class="metric"><div class="value">' + Math.round((contract.progress || 0) * 100) + '%%</div><div class="label">contract progress</div></div>' +
          '</div>' +
          '<div class="tool-list">' + (toolRows || '<div class="empty">No tool activity yet.</div>') + '</div>' +
          '<div class="meta-row"><span>Required tools</span><span class="mono">' + requiredTools.join(', ') + '</span></div>' +
          '<div class="meta-row"><span>Required profiles</span><span class="mono">' + requiredProfiles.join(', ') + '</span></div>' +
          '<div class="meta-row"><span>Required artifacts</span><span class="mono">' + requiredArtifacts.join(', ') + '</span></div>' +
        '</div>';
    }

    function renderPhaseSummary(status) {
      const root = el('phaseSummary');
      const phases = status && status.completed_phases ? status.completed_phases : [];
      const current = status && status.current_phase ? status.current_phase : null;
      const cards = [];
      phases.forEach(name => cards.push('<div class="summary-row"><strong>' + name + '</strong><span class="phase-status">COMPLETED</span></div>'));
      if (current) cards.push('<div class="summary-row"><strong>' + current + '</strong><span class="' + badgeClass(status.status) + '">' + status.status + '</span></div>');
      root.innerHTML = cards.length ? cards.join('') : '<div class="empty">No phase activity yet.</div>';
    }

    function renderFrontier(frontier) {
      const root = el('frontier');
      if (!frontier) {
        root.innerHTML = 'Loading graph frontier...';
        return;
      }
      const recs = (frontier.recommendations || []).slice(0, 6).map(r => '<div class="summary-row"><strong>' + (r.Tool || r.tool) + '</strong><span class="muted">' + ((r.Reason || r.reason || '').trim()) + '</span></div>').join('');
      root.innerHTML =
        '<div class="graph-card">' +
          '<div class="muted" style="margin-bottom:10px">' + (frontier.summary || 'No frontier summary available.') + '</div>' +
          (recs ? '<div class="summary-list">' + recs + '</div>' : '<div class="muted">No tool recommendations right now.</div>') +
        '</div>';
    }

    function renderGraphFilters() {
      const typeSelect = el('graphTypeFilter');
      const searchInput = el('graphSearch');
      const types = ['all'].concat(Array.from(new Set((state.graphNodes || []).map(node => node.type))).sort());
      typeSelect.innerHTML = types.map(type =>
        '<option value="' + type + '" ' + (state.graphType === type ? 'selected' : '') + '>' + (type === 'all' ? 'All node types' : type) + '</option>'
      ).join('');
      searchInput.value = state.graphSearch;
      searchInput.oninput = () => {
        state.graphSearch = searchInput.value || '';
        renderGraphExplorer();
      };
      typeSelect.onchange = () => {
        state.graphType = typeSelect.value;
        renderGraphExplorer();
      };
    }

    function renderGraphExplorer() {
      const root = el('graphExplorer');
      const nodes = (state.graphNodes || []).filter(node => {
        const matchesType = state.graphType === 'all' || node.type === state.graphType;
        const needle = state.graphSearch.trim().toLowerCase();
        const haystack = (String(node.label || '') + ' ' + String(node.id || '')).toLowerCase();
        const matchesSearch = needle === '' || haystack.includes(needle);
        return matchesType && matchesSearch;
      });
      if (state.selectedNodeId && !nodes.find(node => node.id === state.selectedNodeId)) {
        state.selectedNodeId = nodes[0] ? nodes[0].id : null;
      }
      if (nodes.length === 0) {
        root.innerHTML = '<div class="empty">No graph nodes match the current filters.</div>';
        return;
      }
      root.innerHTML = '<div class="node-grid">' + nodes.slice(0, 12).map(node =>
        '<button class="node-card ' + (state.selectedNodeId === node.id ? 'active' : '') + '" data-node-id="' + node.id + '">' +
          '<div class="node-header">' +
            '<strong>' + (node.label || node.id) + '</strong>' +
            '<span class="pill">' + node.type + '</span>' +
          '</div>' +
          '<div class="muted mono">' + node.id + '</div>' +
          (node.is_frontier ? '<div class="tool-row"><span>Frontier candidate</span><span class="phase-status warn">active</span></div>' : '') +
        '</button>'
      ).join('') + '</div>';

      root.querySelectorAll('[data-node-id]').forEach(button => {
        button.addEventListener('click', () => {
          state.selectedNodeId = button.getAttribute('data-node-id');
          renderGraphExplorer();
          renderNodeDetails();
        });
      });

      if (!state.selectedNodeId && nodes[0]) {
        state.selectedNodeId = nodes[0].id;
        renderGraphExplorer();
        renderNodeDetails();
      }
    }

    function renderNodeDetails() {
      const root = el('nodeDetails');
      const node = (state.graphNodes || []).find(item => item.id === state.selectedNodeId);
      if (!node) {
        root.innerHTML = 'Select a node from the graph explorer to inspect its properties and edges.';
        return;
      }

      const edges = (state.graphEdges || []).filter(edge => edge.source === node.id || edge.target === node.id);
      const properties = Object.entries(node.properties || {});
      const propertyRows = properties.length
        ? properties.map(([key, value]) =>
            '<div class="property-row"><strong>' + key + '</strong><span class="mono">' + JSON.stringify(value) + '</span></div>'
          ).join('')
        : '<div class="empty">No node properties recorded.</div>';
      const edgeRows = edges.length
        ? edges.slice(0, 8).map(edge =>
            '<div class="summary-row"><strong>' + edge.type + '</strong><span class="mono">' + edge.source + ' → ' + edge.target + '</span></div>'
          ).join('')
        : '<div class="empty">No connected edges recorded.</div>';

      root.innerHTML =
        '<div class="node-header">' +
          '<strong>' + (node.label || node.id) + '</strong>' +
          '<span class="pill">' + node.type + '</span>' +
        '</div>' +
        '<div class="muted mono">' + node.id + '</div>' +
        '<div class="tool-row"><span>Frontier</span><span class="' + (node.is_frontier ? 'phase-status warn' : 'phase-status') + '">' + (node.is_frontier ? 'yes' : 'no') + '</span></div>' +
        '<div style="margin-top:14px"><strong>Properties</strong></div>' +
        '<div class="property-list">' + propertyRows + '</div>' +
        '<div style="margin-top:14px"><strong>Connected Edges</strong></div>' +
        '<div class="summary-list" style="margin-top:10px">' + edgeRows + '</div>';
    }

    function formatEventPayload(event) {
      const payload = event.payload || {};
      switch (event.type) {
        case 'phase_start':
          return 'Started ' + payload.phase + ' · iteration ' + payload.iteration + '\n' + (payload.objective || '');
        case 'phase_complete':
          return 'Completed ' + payload.phase + ' as ' + payload.status + ' in ' + (payload.duration || 'unknown duration');
        case 'tool_execution':
          return payload.tool + ' · ' + payload.status + '\n' + (payload.summary || '');
        case 'artifact':
          return payload.type + ' from ' + payload.phase + ' · count ' + payload.count;
        case 'graph_update':
          return 'Graph mutation · nodes ' + payload.nodes + ' · edges ' + payload.edges;
        case 'log':
          return (payload.component || 'runtime') + ' · ' + (payload.level || 'info') + '\n' + (payload.message || '');
        default:
          return JSON.stringify(payload, null, 2);
      }
    }

    function renderEventFilters() {
      const root = el('eventFilters');
      root.innerHTML = eventFilters.map(filter =>
        '<button class="tab-button ' + (state.eventFilter === filter ? 'active' : '') + '" data-filter="' + filter + '">' + filter.split('_').join(' ') + '</button>'
      ).join('');
      root.querySelectorAll('[data-filter]').forEach(button => {
        button.addEventListener('click', () => {
          state.eventFilter = button.getAttribute('data-filter');
          renderEventFilters();
          renderEvents();
        });
      });
    }

    function renderEvents() {
      const root = el('events');
      const visibleEvents = (state.eventFilter === 'all'
        ? state.events
        : state.events.filter(event => event.type === state.eventFilter)
      );
      if (visibleEvents.length === 0) {
        root.innerHTML = '<div class="empty">Waiting for live events...</div>';
        return;
      }
      root.innerHTML = visibleEvents.slice(0, 12).map(event =>
        '<div class="log-entry">' +
          '<div style="display:flex;justify-content:space-between;gap:12px;margin-bottom:8px;">' +
            '<span class="event-type ' + event.type + '">' + String(event.type).split('_').join(' ') + '</span>' +
            '<span class="muted mono">' + new Date(event.time).toLocaleTimeString() + '</span>' +
          '</div>' +
          '<pre>' + formatEventPayload(event) + '</pre>' +
        '</div>'
      ).join('');
    }

    async function loadJSON(path) {
      const res = await fetch(path);
      if (!res.ok) throw new Error(path + ' failed');
      return res.json();
    }

    async function loadSnapshot() {
      const [status, phaseDetail, artifacts, tools, frontier, nodes, edges] = await Promise.all([
        loadJSON('/api/pipeline/status'),
        fetch('/api/phase').then(r => r.ok ? r.json() : null),
        loadJSON('/api/artifacts').catch(() => []),
        loadJSON('/api/tools').catch(() => []),
        loadJSON('/api/graph/frontier').catch(() => null),
        loadJSON('/api/graph/nodes').catch(() => []),
        loadJSON('/api/graph/edges').catch(() => []),
      ]);

      el('pipelineName').textContent = status.name || '-';
      el('pipelineStatus').textContent = status.status || '-';
      el('artifactCount').textContent = String(status.artifact_count || 0);
      el('graphNodes').textContent = String(status.graph_nodes || 0);
      el('progressBar').style.width = Math.max(0, Math.min(100, status.progress || 0)) + '%%';

      renderPhaseSummary(status);
      renderCurrentPhase(phaseDetail);
      renderArtifacts(artifacts);
      renderTools(tools);
      renderFrontier(frontier);
      state.graphNodes = nodes;
      state.graphEdges = edges;
      renderGraphFilters();
      renderGraphExplorer();
      renderNodeDetails();
    }

    function connectWS() {
      const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
      const ws = new WebSocket(protocol + '//' + location.host + '/ws/pipeline');
      ws.addEventListener('open', () => {
        el('connectionState').textContent = 'Live';
      });
      ws.addEventListener('close', () => {
        el('connectionState').textContent = 'Reconnecting';
        setTimeout(connectWS, 1500);
      });
      ws.addEventListener('message', (evt) => {
        const chunks = String(evt.data).split('\n').filter(Boolean);
        chunks.forEach(chunk => {
          try {
            const event = JSON.parse(chunk);
            state.events.unshift(event);
          } catch (_) {}
        });
        renderEvents();
        loadSnapshot().catch(console.error);
      });
    }

    loadSnapshot().catch(console.error);
    renderEventFilters();
    connectWS();
  </script>
</body>
</html>`)
}
