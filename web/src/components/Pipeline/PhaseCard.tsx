import { useQuery } from '@tanstack/react-query';
import { fetchPhaseDetail } from '../../api/client';

interface PhaseCardProps {
  phaseName: string;
}

export default function PhaseCard({ phaseName }: PhaseCardProps) {
  const { data: phaseDetail, isLoading } = useQuery({
    queryKey: ['phase-detail', phaseName],
    queryFn: fetchPhaseDetail,
    refetchInterval: 2000,
  });

  if (isLoading || !phaseDetail) {
    return (
      <div className="bg-claw-dark rounded-lg p-6">
        <div className="animate-pulse">
          <div className="h-4 bg-gray-700 rounded w-1/3 mb-4"></div>
          <div className="h-8 bg-gray-700 rounded w-2/3"></div>
        </div>
      </div>
    );
  }

  const contract = phaseDetail.contract;

  return (
    <div className="bg-claw-dark rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h4 className="text-xl font-bold text-white">{phaseDetail.name}</h4>
        <span className={`px-3 py-1 rounded-full text-xs font-semibold ${
          phaseDetail.status === 'RUNNING' ? 'bg-claw-yellow/20 text-claw-yellow' :
          phaseDetail.status === 'COMPLETED' ? 'bg-claw-green/20 text-claw-green' :
          'bg-claw-red/20 text-claw-red'
        }`}>
          {phaseDetail.status}
        </span>
      </div>

      {/* Iteration Progress */}
      <div className="mb-4">
        <div className="flex justify-between text-sm text-gray-400 mb-2">
          <span>Iteration {phaseDetail.iteration}/{phaseDetail.max_iterations}</span>
          <span>{Math.round((phaseDetail.iteration / phaseDetail.max_iterations) * 100)}%</span>
        </div>
        <div className="w-full bg-gray-800 rounded-full h-2">
          <div
            className="bg-claw-blue h-2 rounded-full transition-all duration-300"
            style={{ width: `${(phaseDetail.iteration / phaseDetail.max_iterations) * 100}%` }}
          ></div>
        </div>
      </div>

      {/* Contract Status */}
      {contract && (
        <div className="border-t border-gray-800 pt-4">
          <div className="flex items-center justify-between mb-3">
            <h5 className="text-sm font-semibold text-gray-300">Contract Status</h5>
            {contract.satisfied ? (
              <span className="text-xs text-claw-green flex items-center">
                <svg className="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
                Satisfied
              </span>
            ) : (
              <span className="text-xs text-claw-yellow">In Progress</span>
            )}
          </div>

          <div className="space-y-2">
            <div>
              <p className="text-xs text-gray-400">Required Tools:</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {contract.required_tools.map((tool) => (
                  <span key={tool} className="text-xs bg-gray-800 px-2 py-1 rounded text-gray-300">
                    {tool}
                  </span>
                ))}
              </div>
            </div>
            <div>
              <p className="text-xs text-gray-400">Required Artifacts:</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {contract.required_artifacts.map((artifact) => (
                  <span key={artifact} className="text-xs bg-gray-800 px-2 py-1 rounded text-gray-300">
                    {artifact}
                  </span>
                ))}
              </div>
            </div>
          </div>

          {/* Contract Progress Bar */}
          <div className="mt-3">
            <div className="w-full bg-gray-800 rounded-full h-2">
              <div
                className="bg-claw-green h-2 rounded-full transition-all duration-300"
                style={{ width: `${contract.progress * 100}%` }}
              ></div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
