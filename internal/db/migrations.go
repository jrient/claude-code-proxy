package db

import "fmt"

func (d *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL DEFAULT 'openai',
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 1,
			weight INTEGER NOT NULL DEFAULT 10,
			enabled INTEGER NOT NULL DEFAULT 1,
			health_status TEXT NOT NULL DEFAULT 'unknown',
			last_health_check TEXT,
			config_json TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			key_prefix TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			rate_limit INTEGER NOT NULL DEFAULT 60,
			daily_token_limit INTEGER NOT NULL DEFAULT 0,
			allowed_models TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id INTEGER,
			provider_id INTEGER,
			model TEXT,
			prompt_tokens INTEGER DEFAULT 0,
			completion_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			latency_ms INTEGER DEFAULT 0,
			status_code INTEGER DEFAULT 0,
			error_msg TEXT DEFAULT '',
			stream INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (api_key_id) REFERENCES api_keys(id),
			FOREIGN KEY (provider_id) REFERENCES providers(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_api_key_id ON request_logs(api_key_id)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_provider_id ON request_logs(provider_id)`,
		`CREATE TABLE IF NOT EXISTS stats_hourly (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hour TEXT NOT NULL,
			api_key_id INTEGER,
			provider_id INTEGER,
			model TEXT,
			request_count INTEGER DEFAULT 0,
			total_prompt_tokens INTEGER DEFAULT 0,
			total_completion_tokens INTEGER DEFAULT 0,
			avg_latency_ms REAL DEFAULT 0,
			p99_latency_ms INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			estimated_cost REAL DEFAULT 0,
			UNIQUE(hour, api_key_id, provider_id, model)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stats_hourly_hour ON stats_hourly(hour)`,
		`CREATE TABLE IF NOT EXISTS model_mappings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL,
			source_model TEXT NOT NULL,
			target_model TEXT NOT NULL,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
			UNIQUE(provider_id, source_model)
		)`,
	}

	for i, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}

	return nil
}
