package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCLAWAdapter_Disabled(t *testing.T) {
	cfg := &CLAWConfig{
		Enabled: false,
	}

	adapter, err := NewCLAWAdapter(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.False(t, adapter.IsEnabled())
}

func TestNewCLAWAdapter_Enabled(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &CLAWConfig{
		Enabled:        true,
		Pipeline:       "web_quick",
		PersistenceDir: tempDir,
	}

	adapter, err := NewCLAWAdapter(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.True(t, adapter.IsEnabled())
	assert.NotNil(t, adapter.GetOrchestrator())
	assert.NotNil(t, adapter.GetBlackboard())
}

func TestNewCLAWAdapter_InvalidPipeline(t *testing.T) {
	cfg := &CLAWConfig{
		Enabled:  true,
		Pipeline: "nonexistent_pipeline",
	}

	_, err := NewCLAWAdapter(cfg, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load pipeline")
}

func TestParseTargetFromMessage(t *testing.T) {
	tests := []struct {
		message      string
		expectedTarget string
		expectedType   string
	}{
		{
			message:        "example.com",
			expectedTarget: "example.com",
			expectedType:   "web",
		},
		{
			message:        "web:example.com",
			expectedTarget: "example.com",
			expectedType:   "web",
		},
		{
			message:        "network:192.168.1.0/24",
			expectedTarget: "192.168.1.0/24",
			expectedType:   "network",
		},
		{
			message:        "source:/path/to/code",
			expectedTarget: "/path/to/code",
			expectedType:   "source",
		},
		{
			message:        "binary:/path/to/binary",
			expectedTarget: "/path/to/binary",
			expectedType:   "binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			target, targetType := parseTargetFromMessage(tt.message)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedType, targetType)
		})
	}
}

func TestCLAWAdapter_ProcessMessage_Disabled(t *testing.T) {
	adapter := &CLAWAdapter{enabled: false}

	_, err := adapter.ProcessMessage(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			input:    "this is a very long string that needs truncation",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			input:    "exactly10c",
			maxLen:   10,
			expected: "exactly10c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
