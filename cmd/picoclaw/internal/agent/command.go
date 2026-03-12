package agent

import (
	"github.com/spf13/cobra"
)

func NewAgentCommand() *cobra.Command {
	var (
		message       string
		sessionKey    string
		model         string
		debug         bool
		useTUI        bool
		webUIAddr     string
		autoOpenWebUI bool
		workflowName  string
		target        string
	)

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Interact with the agent directly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return agentCmd(message, sessionKey, model, debug, useTUI, webUIAddr, autoOpenWebUI, workflowName, target)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message (non-interactive mode)")
	cmd.Flags().StringVarP(&sessionKey, "session", "s", "cli:default", "Session key")
	cmd.Flags().StringVarP(&model, "model", "", "", "Model to use")
	cmd.Flags().BoolVar(&useTUI, "tui", false, "Use terminal UI (interactive mode only)")
	cmd.Flags().StringVar(&webUIAddr, "webui", "", "Start embedded local web UI (optionally set address like 127.0.0.1:0 or :8080)")
	cmd.Flags().BoolVar(&autoOpenWebUI, "open-webui", false, "Open the embedded web UI in your browser after startup")
	cmd.Flags().StringVarP(&workflowName, "workflow", "w", "", "Load workflow for guided assessment (e.g., 'network-scan')")
	cmd.Flags().StringVarP(&target, "target", "t", "", "Target for workflow mission (e.g., IP range, domain, URL)")

	return cmd
}
