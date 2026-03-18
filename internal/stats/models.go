package stats

import "time"

// RequestLog represents a single API request log entry
type RequestLog struct {
	ID               int64     `json:"id"`
	APIKeyID         int64     `json:"api_key_id"`
	ProviderID       int64     `json:"provider_id"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	LatencyMs        int64     `json:"latency_ms"`
	StatusCode       int       `json:"status_code"`
	ErrorMsg         string    `json:"error_msg"`
	Stream           bool      `json:"stream"`
	CreatedAt        time.Time `json:"created_at"`
}

// HourlyStat represents aggregated hourly statistics
type HourlyStat struct {
	ID                    int64   `json:"id"`
	Hour                  string  `json:"hour"`
	APIKeyID              int64   `json:"api_key_id"`
	ProviderID            int64   `json:"provider_id"`
	Model                 string  `json:"model"`
	RequestCount          int     `json:"request_count"`
	TotalPromptTokens     int     `json:"total_prompt_tokens"`
	TotalCompletionTokens int     `json:"total_completion_tokens"`
	AvgLatencyMs          float64 `json:"avg_latency_ms"`
	P99LatencyMs          int64   `json:"p99_latency_ms"`
	ErrorCount            int     `json:"error_count"`
	EstimatedCost         float64 `json:"estimated_cost"`
}

// DashboardStats represents the overview dashboard data
type DashboardStats struct {
	TotalRequests    int     `json:"total_requests"`
	TotalTokens      int     `json:"total_tokens"`
	ActiveProviders  int     `json:"active_providers"`
	ErrorRate        float64 `json:"error_rate"`
	AvgLatency       float64 `json:"avg_latency"`
	TotalCost        float64 `json:"total_cost"`
	RequestsToday    int     `json:"requests_today"`
	TokensToday      int     `json:"tokens_today"`
}

// TimeSeriesPoint represents a data point in a time series
type TimeSeriesPoint struct {
	Time     string  `json:"time"`
	Value    float64 `json:"value"`
	Requests int     `json:"requests,omitempty"`
	Tokens   int     `json:"tokens,omitempty"`
	Errors   int     `json:"errors,omitempty"`
	Cost     float64 `json:"cost,omitempty"`
}

// CostEstimator estimates costs based on model and token counts
var ModelCosts = map[string]struct {
	InputPer1M  float64
	OutputPer1M float64
}{
	"claude-sonnet-4-20250514":       {3.0, 15.0},
	"claude-opus-4-20250514":         {15.0, 75.0},
	"claude-haiku-3-5-20241022":      {0.80, 4.0},
	"claude-3-5-sonnet-20241022":     {3.0, 15.0},
	"claude-3-5-haiku-20241022":      {0.80, 4.0},
}

func EstimateCost(model string, promptTokens, completionTokens int) float64 {
	costs, ok := ModelCosts[model]
	if !ok {
		// Default cost estimate
		costs = struct {
			InputPer1M  float64
			OutputPer1M float64
		}{3.0, 15.0}
	}
	return (float64(promptTokens) * costs.InputPer1M / 1_000_000) +
		(float64(completionTokens) * costs.OutputPer1M / 1_000_000)
}
