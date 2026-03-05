import { GraphNode } from '../../api/client';

interface NodeDetailsModalProps {
  node: GraphNode | null;
  onClose: () => void;
}

export default function NodeDetailsModal({ node, onClose }: NodeDetailsModalProps) {
  if (!node) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/70"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-claw-dark rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-800">
          <div>
            <h3 className="text-xl font-bold text-white">{node.label}</h3>
            <div className="flex items-center space-x-2 mt-1">
              <span className="text-sm text-gray-400">Type:</span>
              <span className="text-sm font-semibold text-claw-blue">{node.type}</span>
              {node.is_frontier && (
                <span className="px-2 py-0.5 text-xs font-semibold bg-claw-yellow/20 text-claw-yellow rounded">
                  Frontier
                </span>
              )}
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(80vh-140px)]">
          {/* Node ID */}
          <div className="mb-4">
            <h4 className="text-sm font-semibold text-gray-400 mb-2">Node ID</h4>
            <code className="text-sm text-claw-blue bg-gray-900 px-3 py-1.5 rounded block">
              {node.id}
            </code>
          </div>

          {/* Properties */}
          {Object.keys(node.properties).length > 0 && (
            <div className="mb-4">
              <h4 className="text-sm font-semibold text-gray-400 mb-2">Properties</h4>
              <div className="bg-gray-900 rounded p-4 space-y-2">
                {Object.entries(node.properties).map(([key, value]) => (
                  <div key={key} className="flex items-start">
                    <span className="text-sm text-gray-400 min-w-[120px]">{key}:</span>
                    <span className="text-sm text-white font-mono flex-1">
                      {typeof value === 'object' ? JSON.stringify(value, null, 2) : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Frontier Info */}
          {node.is_frontier && (
            <div className="bg-claw-yellow/10 border border-claw-yellow/30 rounded p-4">
              <div className="flex items-start">
                <svg className="w-5 h-5 text-claw-yellow mr-2 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                </svg>
                <div>
                  <h5 className="text-sm font-semibold text-claw-yellow mb-1">Frontier Node</h5>
                  <p className="text-sm text-gray-300">
                    This entity has unknown properties that need exploration. Tools can be called to discover more information about this node.
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Empty state */}
          {Object.keys(node.properties).length === 0 && !node.is_frontier && (
            <div className="text-center py-8 text-gray-500">
              <p>No additional properties available</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end p-4 border-t border-gray-800">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-claw-blue hover:bg-blue-600 text-white rounded transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
