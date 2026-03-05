import { useState, useEffect } from 'react';
import { useWebSocket, WebSocketEvent } from '../../hooks/useWebSocket';

interface ToolEvent {
  id: string;
  tool: string;
  status: string;
  summary: string;
  timestamp: string;
}

export default function ToolExecutionLog() {
  const [events, setEvents] = useState<ToolEvent[]>([]);

  const { lastMessage } = useWebSocket('ws://localhost:8080/ws/pipeline', {
    onMessage: (event: WebSocketEvent) => {
      if (event.type === 'tool_execution') {
        const toolEvent: ToolEvent = {
          id: `${event.payload.tool}-${Date.now()}`,
          tool: event.payload.tool,
          status: event.payload.status,
          summary: event.payload.summary || '',
          timestamp: event.time,
        };
        setEvents((prev) => [toolEvent, ...prev].slice(0, 50)); // Keep last 50 events
      }
    },
  });

  if (events.length === 0) {
    return (
      <div className="bg-claw-dark rounded-lg p-6 text-center text-gray-400">
        <p>No tool executions yet</p>
        <p className="text-sm text-gray-500 mt-2">Events will appear here as tools are executed</p>
      </div>
    );
  }

  return (
    <div className="bg-claw-dark rounded-lg overflow-hidden">
      <div className="max-h-96 overflow-y-auto">
        <table className="w-full">
          <thead className="bg-gray-900 sticky top-0">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                Time
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                Tool
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                Status
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">
                Summary
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {events.map((event) => (
              <tr key={event.id} className="hover:bg-gray-800/50 transition-colors">
                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-400">
                  {new Date(event.timestamp).toLocaleTimeString()}
                </td>
                <td className="px-4 py-3 whitespace-nowrap">
                  <code className="text-sm text-claw-blue">{event.tool}</code>
                </td>
                <td className="px-4 py-3 whitespace-nowrap">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    event.status === 'completed' ? 'bg-claw-green/20 text-claw-green' :
                    event.status === 'running' ? 'bg-claw-yellow/20 text-claw-yellow' :
                    'bg-claw-red/20 text-claw-red'
                  }`}>
                    {event.status === 'completed' && (
                      <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                      </svg>
                    )}
                    {event.status === 'running' && (
                      <div className="animate-spin mr-1">
                        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      </div>
                    )}
                    {event.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-300 max-w-md truncate">
                  {event.summary || '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
