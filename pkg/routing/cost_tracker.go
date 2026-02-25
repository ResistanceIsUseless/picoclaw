package routing

import (
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// CostTracker tracks token usage and costs across sessions and models
type CostTracker struct {
	mu       sync.RWMutex
	sessions map[string]*SessionCost
}

// SessionCost tracks costs for a single session
type SessionCost struct {
	SessionKey string
	ByModel    map[string]*ModelCost
	ByTier     map[string]*TierCost
	TotalCost  float64
	StartTime  time.Time
	LastUpdate time.Time
}

// ModelCost tracks usage and cost for a specific model
type ModelCost struct {
	ModelName    string
	InputTokens  int
	OutputTokens int
	Calls        int
	TotalCost    float64
	TotalLatency time.Duration
	AvgLatency   time.Duration
}

// TierCost tracks usage and cost for a specific tier
type TierCost struct {
	TierName     string
	InputTokens  int
	OutputTokens int
	Calls        int
	TotalCost    float64
	TotalLatency time.Duration
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		sessions: make(map[string]*SessionCost),
	}
}

// Record records token usage and calculates cost
func (ct *CostTracker) Record(
	sessionKey string,
	modelName string,
	tierName string,
	tierCfg config.TierConfig,
	usage providers.UsageInfo,
	latency time.Duration,
) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Get or create session cost
	session, ok := ct.sessions[sessionKey]
	if !ok {
		session = &SessionCost{
			SessionKey: sessionKey,
			ByModel:    make(map[string]*ModelCost),
			ByTier:     make(map[string]*TierCost),
			StartTime:  time.Now(),
		}
		ct.sessions[sessionKey] = session
	}

	// Get or create model cost
	model, ok := session.ByModel[modelName]
	if !ok {
		model = &ModelCost{
			ModelName: modelName,
		}
		session.ByModel[modelName] = model
	}

	// Get or create tier cost
	tier, ok := session.ByTier[tierName]
	if !ok {
		tier = &TierCost{
			TierName: tierName,
		}
		session.ByTier[tierName] = tier
	}

	// Calculate cost for this call
	inputCost := float64(usage.PromptTokens) / 1_000_000.0 * tierCfg.CostPerM.Input
	outputCost := float64(usage.CompletionTokens) / 1_000_000.0 * tierCfg.CostPerM.Output
	callCost := inputCost + outputCost

	// Update model stats
	model.InputTokens += usage.PromptTokens
	model.OutputTokens += usage.CompletionTokens
	model.Calls++
	model.TotalCost += callCost
	model.TotalLatency += latency
	model.AvgLatency = model.TotalLatency / time.Duration(model.Calls)

	// Update tier stats
	tier.InputTokens += usage.PromptTokens
	tier.OutputTokens += usage.CompletionTokens
	tier.Calls++
	tier.TotalCost += callCost
	tier.TotalLatency += latency

	// Update session totals
	session.TotalCost += callCost
	session.LastUpdate = time.Now()
}

// GetSessionCost returns cost information for a session
func (ct *CostTracker) GetSessionCost(sessionKey string) *SessionCost {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	session, ok := ct.sessions[sessionKey]
	if !ok {
		return nil
	}

	// Return a copy to prevent external mutation
	copy := &SessionCost{
		SessionKey: session.SessionKey,
		ByModel:    make(map[string]*ModelCost),
		ByTier:     make(map[string]*TierCost),
		TotalCost:  session.TotalCost,
		StartTime:  session.StartTime,
		LastUpdate: session.LastUpdate,
	}

	for k, v := range session.ByModel {
		modelCopy := *v
		copy.ByModel[k] = &modelCopy
	}

	for k, v := range session.ByTier {
		tierCopy := *v
		copy.ByTier[k] = &tierCopy
	}

	return copy
}

// GetTotalCost returns the total cost across all sessions
func (ct *CostTracker) GetTotalCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	total := 0.0
	for _, session := range ct.sessions {
		total += session.TotalCost
	}
	return total
}

// FormatSessionReport generates a human-readable cost report for a session
func (ct *CostTracker) FormatSessionReport(sessionKey string) string {
	session := ct.GetSessionCost(sessionKey)
	if session == nil {
		return "No cost data for session"
	}

	duration := session.LastUpdate.Sub(session.StartTime)

	report := fmt.Sprintf("Session Cost Report\n")
	report += fmt.Sprintf("==================\n")
	report += fmt.Sprintf("Session: %s\n", sessionKey)
	report += fmt.Sprintf("Duration: %s\n", duration.Round(time.Second))
	report += fmt.Sprintf("Total Cost: $%.4f\n\n", session.TotalCost)

	report += fmt.Sprintf("By Tier:\n")
	report += fmt.Sprintf("--------\n")
	for tierName, tier := range session.ByTier {
		report += fmt.Sprintf("  %s:\n", tierName)
		report += fmt.Sprintf("    Calls: %d\n", tier.Calls)
		report += fmt.Sprintf("    Input tokens: %d\n", tier.InputTokens)
		report += fmt.Sprintf("    Output tokens: %d\n", tier.OutputTokens)
		report += fmt.Sprintf("    Cost: $%.4f\n", tier.TotalCost)
		if tier.Calls > 0 {
			avgLatency := tier.TotalLatency / time.Duration(tier.Calls)
			report += fmt.Sprintf("    Avg latency: %s\n", avgLatency.Round(time.Millisecond))
		}
		report += fmt.Sprintf("\n")
	}

	report += fmt.Sprintf("By Model:\n")
	report += fmt.Sprintf("---------\n")
	for modelName, model := range session.ByModel {
		report += fmt.Sprintf("  %s:\n", modelName)
		report += fmt.Sprintf("    Calls: %d\n", model.Calls)
		report += fmt.Sprintf("    Input tokens: %d\n", model.InputTokens)
		report += fmt.Sprintf("    Output tokens: %d\n", model.OutputTokens)
		report += fmt.Sprintf("    Cost: $%.4f\n", model.TotalCost)
		report += fmt.Sprintf("    Avg latency: %s\n", model.AvgLatency.Round(time.Millisecond))
		report += fmt.Sprintf("\n")
	}

	return report
}

// Reset clears all cost tracking data
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.sessions = make(map[string]*SessionCost)
}
