package stats

import (
	"database/sql"
	"log"
	"time"
)

// Collector asynchronously collects request statistics
type Collector struct {
	db   *sql.DB
	ch   chan *RequestLog
	stop chan struct{}
}

func NewCollector(db *sql.DB) *Collector {
	return &Collector{
		db:   db,
		ch:   make(chan *RequestLog, 1000),
		stop: make(chan struct{}),
	}
}

func (c *Collector) Start() {
	go c.worker()
	go c.aggregateWorker()
}

func (c *Collector) Stop() {
	close(c.stop)
}

// Record sends a log entry to the async collector
func (c *Collector) Record(entry *RequestLog) {
	select {
	case c.ch <- entry:
	default:
		log.Println("[stats] collector channel full, dropping entry")
	}
}

func (c *Collector) worker() {
	for {
		select {
		case entry := <-c.ch:
			c.insert(entry)
		case <-c.stop:
			// Drain remaining entries
			for {
				select {
				case entry := <-c.ch:
					c.insert(entry)
				default:
					return
				}
			}
		}
	}
}

func (c *Collector) insert(entry *RequestLog) {
	streamInt := 0
	if entry.Stream {
		streamInt = 1
	}
	_, err := c.db.Exec(
		`INSERT INTO request_logs (api_key_id, provider_id, model, prompt_tokens, completion_tokens, total_tokens, latency_ms, status_code, error_msg, stream)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.APIKeyID, entry.ProviderID, entry.Model,
		entry.PromptTokens, entry.CompletionTokens, entry.TotalTokens,
		entry.LatencyMs, entry.StatusCode, entry.ErrorMsg, streamInt,
	)
	if err != nil {
		log.Printf("[stats] insert error: %v", err)
	}
}

// aggregateWorker periodically aggregates raw logs into hourly stats
func (c *Collector) aggregateWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.aggregate()
		case <-c.stop:
			return
		}
	}
}

func (c *Collector) aggregate() {
	// Aggregate the current hour and previous hour
	now := time.Now().UTC()
	hours := []string{
		now.Truncate(time.Hour).Format("2006-01-02T15:00:00Z"),
		now.Add(-time.Hour).Truncate(time.Hour).Format("2006-01-02T15:00:00Z"),
	}

	for _, hour := range hours {
		nextHour, _ := time.Parse("2006-01-02T15:00:00Z", hour)
		nextHourStr := nextHour.Add(time.Hour).Format("2006-01-02T15:00:00Z")

		rows, err := c.db.Query(`
			SELECT api_key_id, provider_id, model,
				COUNT(*) as req_count,
				COALESCE(SUM(prompt_tokens), 0),
				COALESCE(SUM(completion_tokens), 0),
				COALESCE(AVG(latency_ms), 0),
				COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0)
			FROM request_logs
			WHERE created_at >= ? AND created_at < ?
			GROUP BY api_key_id, provider_id, model
		`, hour, nextHourStr)

		if err != nil {
			log.Printf("[stats] aggregate query error: %v", err)
			continue
		}

		for rows.Next() {
			var apiKeyID, providerID int64
			var model string
			var reqCount, promptTokens, completionTokens, errorCount int
			var avgLatency float64

			if err := rows.Scan(&apiKeyID, &providerID, &model, &reqCount, &promptTokens, &completionTokens, &avgLatency, &errorCount); err != nil {
				log.Printf("[stats] aggregate scan error: %v", err)
				continue
			}

			cost := EstimateCost(model, promptTokens, completionTokens)

			_, err := c.db.Exec(`
				INSERT INTO stats_hourly (hour, api_key_id, provider_id, model, request_count, total_prompt_tokens, total_completion_tokens, avg_latency_ms, error_count, estimated_cost)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(hour, api_key_id, provider_id, model)
				DO UPDATE SET request_count=excluded.request_count, total_prompt_tokens=excluded.total_prompt_tokens,
					total_completion_tokens=excluded.total_completion_tokens, avg_latency_ms=excluded.avg_latency_ms,
					error_count=excluded.error_count, estimated_cost=excluded.estimated_cost
			`, hour, apiKeyID, providerID, model, reqCount, promptTokens, completionTokens, avgLatency, errorCount, cost)

			if err != nil {
				log.Printf("[stats] aggregate upsert error: %v", err)
			}
		}
		rows.Close()
	}
}
