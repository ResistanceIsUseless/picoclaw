package registry

import (
	"context"
	"fmt"
)

// ExecuteMockTool executes mock tools for testing
// Returns mock output that parsers can process
func ExecuteMockTool(ctx context.Context, toolName string, args map[string]interface{}) ([]byte, error) {
	switch toolName {
	case "mock_subfinder":
		// Return empty output - the parser will create the mock artifact
		return []byte{}, nil

	case "mock_recon":
		// Return empty output - the parser will create the mock artifact
		return []byte{}, nil

	case "mock_scan":
		// Return empty output - the parser will create the mock artifact
		return []byte{}, nil

	default:
		return nil, fmt.Errorf("unknown mock tool: %s", toolName)
	}
}
