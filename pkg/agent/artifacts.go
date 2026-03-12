package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools/profiles"
)

func buildCompactBlackboardSummary(bb *blackboard.Blackboard) string {
	if bb == nil {
		return ""
	}

	summary := strings.TrimSpace(bb.Summary())
	if summary == "" || summary == "Blackboard is empty" {
		return ""
	}

	if len(summary) > 1200 {
		return summary[:1200] + "\n... (truncated)"
	}

	return summary
}

func (al *AgentLoop) publishToolArtifact(ctx context.Context, toolCallName string, args map[string]any, rawOutput string) string {
	if al.blackboard == nil || al.toolMetadata == nil || rawOutput == "" {
		return ""
	}

	resolvedToolName := resolveArtifactToolName(toolCallName, args)
	if resolvedToolName == "" {
		return ""
	}

	profile, ok := profiles.ResolveToolProfile(resolvedToolName)
	if !ok || profile.ArtifactTool == "" {
		return ""
	}

	toolDef, err := al.toolMetadata.Get(profile.ArtifactTool)
	if err != nil || toolDef.Parser == nil {
		return ""
	}

	parsed, err := toolDef.Parser(profile.ArtifactTool, []byte(rawOutput))
	if err != nil {
		logger.DebugCF("agent", "Structured artifact parsing skipped", map[string]any{
			"tool":  profile.ArtifactTool,
			"error": err.Error(),
		})
		return ""
	}

	artifact, ok := parsed.(blackboard.Artifact)
	if !ok || artifact == nil {
		return ""
	}

	if err := al.blackboard.Publish(ctx, artifact); err != nil {
		logger.WarnCF("agent", "Failed to publish artifact from tool result", map[string]any{
			"tool":  profile.ArtifactTool,
			"error": err.Error(),
		})
		return ""
	}

	logger.InfoCF("agent", "Published structured artifact from tool result", map[string]any{
		"tool":          profile.ArtifactTool,
		"profile":       profile.Name,
		"artifact_type": artifact.Type(),
	})

	return fmt.Sprintf("[Structured artifact recorded: %s via %s profile]", artifact.Type(), profile.Name)
}

func resolveArtifactToolName(toolCallName string, args map[string]any) string {
	if toolCallName == "" {
		return ""
	}

	if toolCallName != "exec" {
		return toolCallName
	}

	command, _ := args["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	first := strings.Trim(fields[0], `"'`)
	if first == "" {
		return ""
	}

	return filepath.Base(first)
}
