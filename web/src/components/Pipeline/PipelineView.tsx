import { useQuery } from '@tanstack/react-query';
import { useWebSocket } from '../../hooks/useWebSocket';
import { fetchPipelineStatus, fetchPhaseDetail, PipelineStatus } from '../../api/client';
import { useState, useEffect } from 'react';
import PhaseCard from './PhaseCard';
import ToolExecutionLog from './ToolExecutionLog';

export default function PipelineView() {
  const [pipelineStatus, setPipelineStatus] = useState<PipelineStatus | null>(null);

  // Poll pipeline status
  const { data: statusData, isLoading } = useQuery({
    queryKey: ['pipeline-status'],
    queryFn: fetchPipelineStatus,
    refetchInterval: 2000, // Poll every 2 seconds
  });

  // WebSocket for real-time updates
  const { lastMessage, isConnected } = useWebSocket('ws://localhost:8080/ws/pipeline', {
    onMessage: (event) => {
      console.log('WebSocket event:', event);
      // Trigger refetch on events
      if (event.type === 'phase_start' || event.type === 'phase_complete' || event.type === 'tool_execution') {
        // Status will be refetched via polling
      }
    },
  });

  useEffect(() => {
    if (statusData) {
      setPipelineStatus(statusData);
    }
  }, [statusData]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-claw-blue mx-auto"></div>
          <p className="mt-4 text-gray-400">Loading pipeline...</p>
        </div>
      </div>
    );
  }

  if (!pipelineStatus) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <p className="text-gray-400">No pipeline running</p>
          <p className="text-sm text-gray-500 mt-2">Start a pipeline with test-claw</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col p-6 overflow-auto">
      {/* Status Header */}
      <div className="bg-claw-dark rounded-lg p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-2xl font-bold text-white">{pipelineStatus.name}</h2>
            <p className="text-gray-400 mt-1">
              Status: <span className={`font-semibold ${
                pipelineStatus.status === 'running' ? 'text-claw-yellow' :
                pipelineStatus.status === 'completed' ? 'text-claw-green' :
                'text-claw-red'
              }`}>{pipelineStatus.status}</span>
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-claw-green' : 'bg-gray-600'}`}></div>
            <span className="text-sm text-gray-400">
              {isConnected ? 'Connected' : 'Disconnected'}
            </span>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="w-full bg-gray-800 rounded-full h-4 mb-2">
          <div
            className="bg-claw-blue h-4 rounded-full transition-all duration-500"
            style={{ width: `${pipelineStatus.progress * 100}%` }}
          ></div>
        </div>
        <div className="flex justify-between text-sm text-gray-400">
          <span>Progress: {Math.round(pipelineStatus.progress * 100)}%</span>
          <span>
            Phases: {pipelineStatus.completed_phases.length} completed
          </span>
        </div>
      </div>

      {/* Current Phase */}
      {pipelineStatus.current_phase && (
        <div className="mb-6">
          <h3 className="text-lg font-semibold text-white mb-3">Current Phase</h3>
          <PhaseCard phaseName={pipelineStatus.current_phase} />
        </div>
      )}

      {/* Tool Execution Log */}
      <div className="flex-1">
        <h3 className="text-lg font-semibold text-white mb-3">Recent Activity</h3>
        <ToolExecutionLog />
      </div>

      {/* Statistics */}
      <div className="grid grid-cols-3 gap-4 mt-6">
        <div className="bg-claw-dark rounded-lg p-4">
          <p className="text-gray-400 text-sm">Artifacts</p>
          <p className="text-2xl font-bold text-white mt-1">{pipelineStatus.artifact_count}</p>
        </div>
        <div className="bg-claw-dark rounded-lg p-4">
          <p className="text-gray-400 text-sm">Graph Nodes</p>
          <p className="text-2xl font-bold text-white mt-1">{pipelineStatus.graph_nodes}</p>
        </div>
        <div className="bg-claw-dark rounded-lg p-4">
          <p className="text-gray-400 text-sm">Uptime</p>
          <p className="text-2xl font-bold text-white mt-1">
            {new Date(pipelineStatus.start_time).toLocaleTimeString()}
          </p>
        </div>
      </div>
    </div>
  );
}
