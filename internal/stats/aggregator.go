package stats

import (
	"database/sql"
	"fmt"
)

// Aggregator provides query methods for statistics
type Aggregator struct {
	db *sql.DB
}

func NewAggregator(db *sql.DB) *Aggregator {
	return &Aggregator{db: db}
}

// GetDashboardStats returns overview statistics
func (a *Aggregator) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Total requests and tokens
	err := a.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total_tokens), 0)
		FROM request_logs
	`).Scan(&stats.TotalRequests, &stats.TotalTokens)
	if err != nil {
		return nil, err
	}

	// Requests today
	err = a.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total_tokens), 0)
		FROM request_logs
		WHERE created_at >= date('now')
	`).Scan(&stats.RequestsToday, &stats.TokensToday)
	if err != nil {
		return nil, err
	}

	// Error rate
	var errorCount int
	a.db.QueryRow(`SELECT COUNT(*) FROM request_logs WHERE status_code >= 400`).Scan(&errorCount)
	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalRequests) * 100
	}

	// Average latency
	a.db.QueryRow(`SELECT COALESCE(AVG(latency_ms), 0) FROM request_logs`).Scan(&stats.AvgLatency)

	// Estimated total cost
	rows, err := a.db.Query(`SELECT model, COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0) FROM request_logs GROUP BY model`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var prompt, completion int
			rows.Scan(&model, &prompt, &completion)
			stats.TotalCost += EstimateCost(model, prompt, completion)
		}
	}

	return stats, nil
}

// GetTimeSeries returns time series data for a given period
func (a *Aggregator) GetTimeSeries(period string, days int, apiKeyID, providerID int64) ([]TimeSeriesPoint, error) {
	var timeFormat, groupBy string
	switch period {
	case "hour":
		timeFormat = "%Y-%m-%d %H:00"
		groupBy = "strftime('%Y-%m-%d %H:00', created_at)"
	case "day":
		timeFormat = "%Y-%m-%d"
		groupBy = "strftime('%Y-%m-%d', created_at)"
	default:
		timeFormat = "%Y-%m-%d %H:00"
		groupBy = "strftime('%Y-%m-%d %H:00', created_at)"
	}
	_ = timeFormat

	query := fmt.Sprintf(`
		SELECT %s as time_bucket,
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as tokens,
			COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0) as errors
		FROM request_logs
		WHERE created_at >= datetime('now', '-%d days')
	`, groupBy, days)

	args := []interface{}{}
	if apiKeyID > 0 {
		query += " AND api_key_id = ?"
		args = append(args, apiKeyID)
	}
	if providerID > 0 {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}

	query += fmt.Sprintf(" GROUP BY %s ORDER BY %s", groupBy, groupBy)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.Time, &p.Requests, &p.Tokens, &p.Errors); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, nil
}

// GetModelStats returns per-model statistics
func (a *Aggregator) GetModelStats(days int) ([]map[string]interface{}, error) {
	rows, err := a.db.Query(`
		SELECT model,
			COUNT(*) as requests,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM request_logs
		WHERE created_at >= datetime('now', ? || ' days')
		GROUP BY model
		ORDER BY requests DESC
	`, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var model string
		var requests, promptTokens, completionTokens int
		var avgLatency float64
		rows.Scan(&model, &requests, &promptTokens, &completionTokens, &avgLatency)
		cost := EstimateCost(model, promptTokens, completionTokens)
		results = append(results, map[string]interface{}{
			"model":             model,
			"requests":          requests,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"avg_latency":       avgLatency,
			"estimated_cost":    cost,
		})
	}
	return results, nil
}

// GetRecentLogs returns recent request logs
func (a *Aggregator) GetRecentLogs(limit, offset int) ([]RequestLog, int, error) {
	var total int
	a.db.QueryRow(`SELECT COUNT(*) FROM request_logs`).Scan(&total)

	rows, err := a.db.Query(`
		SELECT id, api_key_id, provider_id, model, prompt_tokens, completion_tokens, total_tokens,
			latency_ms, status_code, error_msg, stream, created_at
		FROM request_logs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var l RequestLog
		var createdAt string
		var stream int
		rows.Scan(&l.ID, &l.APIKeyID, &l.ProviderID, &l.Model, &l.PromptTokens,
			&l.CompletionTokens, &l.TotalTokens, &l.LatencyMs, &l.StatusCode,
			&l.ErrorMsg, &stream, &createdAt)
		l.Stream = stream == 1
		logs = append(logs, l)
	}
	return logs, total, nil
}
