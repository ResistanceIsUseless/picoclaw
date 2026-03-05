import { useQuery } from '@tanstack/react-query';
import { fetchTools } from '../../api/client';

export default function ToolsView() {
  const { data: tools, isLoading } = useQuery({
    queryKey: ['tools'],
    queryFn: fetchTools,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-claw-blue"></div>
      </div>
    );
  }

  const toolsByTier = tools?.reduce((acc, tool) => {
    const tier = tool.tier || 'unknown';
    if (!acc[tier]) acc[tier] = [];
    acc[tier].push(tool);
    return acc;
  }, {} as Record<string, typeof tools>);

  const tierColors = {
    '-1': { bg: 'bg-purple-900/20', text: 'text-purple-400', name: 'Orchestrator' },
    '0': { bg: 'bg-gray-900/20', text: 'text-gray-400', name: 'Hardwired' },
    '1': { bg: 'bg-green-900/20', text: 'text-green-400', name: 'Auto-approve' },
    '2': { bg: 'bg-yellow-900/20', text: 'text-yellow-400', name: 'Human Approval' },
    '3': { bg: 'bg-red-900/20', text: 'text-red-400', name: 'Banned' },
  };

  return (
    <div className="h-full overflow-auto p-6">
      <div className="max-w-6xl mx-auto">
        <div className="mb-8">
          <h2 className="text-3xl font-bold text-white mb-2">Tool Registry</h2>
          <p className="text-gray-400">
            Security tools available to CLAW. Tools are organized by security tier.
          </p>
        </div>

        {/* Tier Legend */}
        <div className="bg-claw-dark rounded-lg p-4 mb-6">
          <h3 className="text-sm font-semibold text-gray-300 mb-3">Security Tiers</h3>
          <div className="grid grid-cols-5 gap-3">
            {Object.entries(tierColors).map(([tier, config]) => (
              <div key={tier} className={`${config.bg} rounded px-3 py-2`}>
                <p className="text-xs text-gray-400">Tier {tier}</p>
                <p className={`text-sm font-medium ${config.text}`}>{config.name}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Tools by Tier */}
        {toolsByTier && Object.entries(toolsByTier).map(([tier, tierTools]) => {
          const config = tierColors[tier as keyof typeof tierColors] || tierColors['0'];
          return (
            <div key={tier} className="mb-8">
              <div className="flex items-center mb-4">
                <h3 className={`text-xl font-bold ${config.text}`}>
                  Tier {tier}: {config.name}
                </h3>
                <span className="ml-3 text-sm text-gray-500">
                  ({tierTools?.length || 0} tools)
                </span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {tierTools?.map((tool) => (
                  <div
                    key={tool.name}
                    className="bg-claw-dark rounded-lg p-4 hover:ring-2 hover:ring-claw-blue transition-all"
                  >
                    <div className="flex items-start justify-between mb-2">
                      <code className="text-sm font-semibold text-claw-blue">
                        {tool.name}
                      </code>
                      <span className={`text-xs px-2 py-1 rounded ${config.bg} ${config.text}`}>
                        T{tier}
                      </span>
                    </div>
                    <p className="text-sm text-gray-400 line-clamp-2">
                      {tool.description}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          );
        })}

        {(!tools || tools.length === 0) && (
          <div className="bg-claw-dark rounded-lg p-12 text-center">
            <p className="text-gray-400">No tools registered</p>
          </div>
        )}
      </div>
    </div>
  );
}
