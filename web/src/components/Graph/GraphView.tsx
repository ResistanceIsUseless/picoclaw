import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useWebSocket } from '../../hooks/useWebSocket';
import { fetchGraphNodes, fetchGraphEdges, GraphNode } from '../../api/client';
import ForceGraph from './ForceGraph';
import NodeDetailsModal from './NodeDetailsModal';
import GraphLegend from './GraphLegend';

export default function GraphView() {
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [searchTerm, setSearchTerm] = useState('');

  // Fetch graph data
  const { data: nodes = [], isLoading: nodesLoading, refetch: refetchNodes } = useQuery({
    queryKey: ['graph-nodes'],
    queryFn: fetchGraphNodes,
    refetchInterval: 5000, // Refresh every 5 seconds
  });

  const { data: edges = [], isLoading: edgesLoading } = useQuery({
    queryKey: ['graph-edges'],
    queryFn: fetchGraphEdges,
    refetchInterval: 5000,
  });

  // WebSocket for real-time updates
  const { isConnected } = useWebSocket('ws://localhost:8080/ws/graph', {
    onMessage: (event) => {
      if (event.type === 'graph_update') {
        console.log('Graph updated:', event.payload);
        refetchNodes();
      }
    },
  });

  // Filter nodes by search term
  const filteredNodes = nodes.filter(node =>
    node.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
    node.type.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const isLoading = nodesLoading || edgesLoading;

  // Empty state
  if (!isLoading && nodes.length === 0) {
    return (
      <div className="h-full flex items-center justify-center p-6">
        <div className="text-center max-w-2xl">
          <div className="bg-claw-dark rounded-lg p-12">
            <svg
              className="w-24 h-24 mx-auto text-gray-600 mb-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1}
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
              />
            </svg>
            <h2 className="text-2xl font-bold text-white mb-3">No Graph Data Yet</h2>
            <p className="text-gray-400 mb-4">
              The knowledge graph will populate as CLAW discovers entities and relationships.
            </p>
            <p className="text-sm text-gray-500">
              Start a pipeline execution to see the graph in action.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-claw-blue mx-auto"></div>
          <p className="mt-4 text-gray-400">Loading graph...</p>
        </div>
      </div>
    );
  }

  // Stats
  const frontierCount = nodes.filter(n => n.is_frontier).length;
  const entityTypes = Array.from(new Set(nodes.map(n => n.type)));

  return (
    <div className="h-full flex flex-col">
      {/* Toolbar */}
      <div className="bg-claw-dark border-b border-gray-800 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-6">
            <div>
              <h2 className="text-xl font-bold text-white">Knowledge Graph</h2>
              <p className="text-sm text-gray-400">
                {nodes.length} nodes, {edges.length} edges
                {frontierCount > 0 && (
                  <span className="text-claw-yellow ml-2">
                    • {frontierCount} frontier
                  </span>
                )}
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-claw-green' : 'bg-gray-600'}`}></div>
              <span className="text-sm text-gray-400">
                {isConnected ? 'Live' : 'Disconnected'}
              </span>
            </div>
          </div>

          {/* Search */}
          <div className="flex items-center space-x-4">
            <div className="relative">
              <input
                type="text"
                placeholder="Search nodes..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="bg-gray-900 text-white px-4 py-2 pl-10 rounded-lg focus:outline-none focus:ring-2 focus:ring-claw-blue"
              />
              <svg
                className="w-5 h-5 text-gray-400 absolute left-3 top-2.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {/* Graph Container */}
      <div className="flex-1 relative">
        <ForceGraph
          nodes={searchTerm ? filteredNodes : nodes}
          edges={edges}
          onNodeClick={setSelectedNode}
        />

        {/* Legend */}
        <div className="absolute bottom-4 left-4">
          <GraphLegend entityTypes={entityTypes} frontierCount={frontierCount} />
        </div>

        {/* Instructions */}
        <div className="absolute top-4 right-4 bg-claw-dark/90 rounded-lg p-3 text-sm text-gray-400 max-w-xs">
          <p className="font-semibold text-white mb-1">Controls</p>
          <ul className="space-y-1 text-xs">
            <li>• <span className="text-gray-300">Drag</span> to pan</li>
            <li>• <span className="text-gray-300">Scroll</span> to zoom</li>
            <li>• <span className="text-gray-300">Click node</span> for details</li>
            <li>• <span className="text-gray-300">Drag node</span> to reposition</li>
          </ul>
        </div>
      </div>

      {/* Node Details Modal */}
      <NodeDetailsModal
        node={selectedNode}
        onClose={() => setSelectedNode(null)}
      />
    </div>
  );
}
