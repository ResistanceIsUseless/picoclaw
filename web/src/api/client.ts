// API Client for CLAW backend

const API_BASE = '/api';

export interface SystemStatus {
  status: string;
  version: string;
  uptime: string;
  active_clients: number;
  timestamp: string;
}

export interface PipelineStatus {
  name: string;
  status: string;
  current_phase: string;
  completed_phases: string[];
  progress: number;
  start_time: string;
  artifact_count: number;
  graph_nodes: number;
}

export interface PhaseDetail {
  name: string;
  status: string;
  iteration: number;
  max_iterations: number;
  dag_state: DAGStateView;
  contract: ContractView;
}

export interface DAGStateView {
  phase_name: string;
  tools: ToolCallView[];
  progress: number;
}

export interface ToolCallView {
  id: string;
  name: string;
  status: string;
  summary?: string;
  started: string;
  ended?: string;
}

export interface ContractView {
  satisfied: boolean;
  required_tools: string[];
  required_artifacts: string[];
  progress: number;
  min_iterations: number;
  max_iterations: number;
}

export interface GraphNode {
  id: string;
  type: string;
  label: string;
  properties: Record<string, any>;
  is_frontier: boolean;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: string;
  properties: Record<string, any>;
}

export interface Artifact {
  id: string;
  type: string;
  phase: string;
  domain: string;
  created_at: string;
  data: Record<string, any>;
}

export interface ToolInfo {
  name: string;
  description: string;
  tier: string;
}

// API Methods

export async function fetchSystemStatus(): Promise<SystemStatus> {
  const response = await fetch(`${API_BASE}/status`);
  if (!response.ok) throw new Error('Failed to fetch system status');
  return response.json();
}

export async function fetchPipelineStatus(): Promise<PipelineStatus> {
  const response = await fetch(`${API_BASE}/pipeline/status`);
  if (!response.ok) throw new Error('Failed to fetch pipeline status');
  return response.json();
}

export async function fetchPhaseDetail(): Promise<PhaseDetail> {
  const response = await fetch(`${API_BASE}/phase`);
  if (!response.ok) throw new Error('Failed to fetch phase detail');
  return response.json();
}

export async function fetchArtifacts(params?: {
  phase?: string;
  type?: string;
}): Promise<Artifact[]> {
  const query = new URLSearchParams(params as any).toString();
  const url = query ? `${API_BASE}/artifacts?${query}` : `${API_BASE}/artifacts`;
  const response = await fetch(url);
  if (!response.ok) throw new Error('Failed to fetch artifacts');
  return response.json();
}

export async function fetchGraphNodes(): Promise<GraphNode[]> {
  const response = await fetch(`${API_BASE}/graph/nodes`);
  if (!response.ok) throw new Error('Failed to fetch graph nodes');
  return response.json();
}

export async function fetchGraphEdges(): Promise<GraphEdge[]> {
  const response = await fetch(`${API_BASE}/graph/edges`);
  if (!response.ok) throw new Error('Failed to fetch graph edges');
  return response.json();
}

export async function fetchTools(): Promise<ToolInfo[]> {
  const response = await fetch(`${API_BASE}/tools`);
  if (!response.ok) throw new Error('Failed to fetch tools');
  return response.json();
}
