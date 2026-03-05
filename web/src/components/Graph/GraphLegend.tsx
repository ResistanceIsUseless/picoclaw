interface GraphLegendProps {
  entityTypes: string[];
  frontierCount: number;
}

const entityTypeColors: Record<string, string> = {
  domain: '#3b82f6',
  subdomain: '#60a5fa',
  ip: '#10b981',
  service: '#f97316',
  endpoint: '#8b5cf6',
  vulnerability: '#ef4444',
};

const entityTypeLabels: Record<string, string> = {
  domain: 'Domain',
  subdomain: 'Subdomain',
  ip: 'IP Address',
  service: 'Service',
  endpoint: 'Endpoint',
  vulnerability: 'Vulnerability',
};

export default function GraphLegend({ entityTypes, frontierCount }: GraphLegendProps) {
  return (
    <div className="bg-claw-dark/90 rounded-lg p-4 backdrop-blur-sm">
      <h4 className="text-sm font-semibold text-white mb-3">Legend</h4>
      <div className="space-y-2">
        {/* Entity Types */}
        {entityTypes.map((type) => (
          <div key={type} className="flex items-center space-x-2">
            <div
              className="w-3 h-3 rounded-full"
              style={{ backgroundColor: entityTypeColors[type] || '#6b7280' }}
            />
            <span className="text-xs text-gray-300">
              {entityTypeLabels[type] || type}
            </span>
          </div>
        ))}

        {/* Frontier */}
        {frontierCount > 0 && (
          <>
            <div className="border-t border-gray-700 pt-2 mt-2" />
            <div className="flex items-center space-x-2">
              <div
                className="w-3 h-3 rounded-full"
                style={{
                  backgroundColor: '#f59e0b',
                  boxShadow: '0 0 6px #f59e0b',
                }}
              />
              <span className="text-xs text-gray-300">
                Frontier ({frontierCount})
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-1 pl-5">
              Unknown properties to explore
            </p>
          </>
        )}
      </div>
    </div>
  );
}
