package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type APIKey struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	KeyPrefix       string    `json:"key_prefix"`
	Enabled         bool      `json:"enabled"`
	RateLimit       int       `json:"rate_limit"`
	DailyTokenLimit int       `json:"daily_token_limit"`
	AllowedModels   string    `json:"allowed_models"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type APIKeyManager struct {
	db *sql.DB
}

func NewAPIKeyManager(db *sql.DB) *APIKeyManager {
	return &APIKeyManager{db: db}
}

// GenerateKey creates a new virtual API key
// Returns the full key (only shown once) and the APIKey record
func (m *APIKeyManager) GenerateKey(name string, rateLimit, dailyTokenLimit int, allowedModels string) (string, *APIKey, error) {
	// Generate random key: ccp-<32 hex chars>
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, fmt.Errorf("generate random bytes: %w", err)
	}

	fullKey := "ccp-" + hex.EncodeToString(bytes)
	prefix := fullKey[:11] + "..." // ccp-xxxx...
	hash := hashKey(fullKey)

	result, err := m.db.Exec(
		`INSERT INTO api_keys (name, key_hash, key_prefix, enabled, rate_limit, daily_token_limit, allowed_models)
		 VALUES (?, ?, ?, 1, ?, ?, ?)`,
		name, hash, prefix, rateLimit, dailyTokenLimit, allowedModels,
	)
	if err != nil {
		return "", nil, fmt.Errorf("insert api key: %w", err)
	}

	id, _ := result.LastInsertId()
	key := &APIKey{
		ID:              id,
		Name:            name,
		KeyPrefix:       prefix,
		Enabled:         true,
		RateLimit:       rateLimit,
		DailyTokenLimit: dailyTokenLimit,
		AllowedModels:   allowedModels,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	return fullKey, key, nil
}

// ValidateKey checks if a key is valid and returns the associated APIKey
func (m *APIKeyManager) ValidateKey(key string) (*APIKey, error) {
	hash := hashKey(key)

	var ak APIKey
	var createdAt, updatedAt string
	err := m.db.QueryRow(
		`SELECT id, name, key_prefix, enabled, rate_limit, daily_token_limit, allowed_models, created_at, updated_at
		 FROM api_keys WHERE key_hash = ?`,
		hash,
	).Scan(&ak.ID, &ak.Name, &ak.KeyPrefix, &ak.Enabled, &ak.RateLimit, &ak.DailyTokenLimit, &ak.AllowedModels, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid api key")
	}
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}

	ak.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	ak.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	if !ak.Enabled {
		return nil, fmt.Errorf("api key is disabled")
	}

	return &ak, nil
}

// ListKeys returns all API keys
func (m *APIKeyManager) ListKeys() ([]APIKey, error) {
	rows, err := m.db.Query(
		`SELECT id, name, key_prefix, enabled, rate_limit, daily_token_limit, allowed_models, created_at, updated_at
		 FROM api_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var ak APIKey
		var createdAt, updatedAt string
		if err := rows.Scan(&ak.ID, &ak.Name, &ak.KeyPrefix, &ak.Enabled, &ak.RateLimit, &ak.DailyTokenLimit, &ak.AllowedModels, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		ak.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		ak.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		keys = append(keys, ak)
	}
	return keys, nil
}

// UpdateKey updates an API key
func (m *APIKeyManager) UpdateKey(id int64, name string, enabled bool, rateLimit, dailyTokenLimit int, allowedModels string) error {
	_, err := m.db.Exec(
		`UPDATE api_keys SET name=?, enabled=?, rate_limit=?, daily_token_limit=?, allowed_models=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		name, enabled, rateLimit, dailyTokenLimit, allowedModels, id,
	)
	return err
}

// DeleteKey deletes an API key
func (m *APIKeyManager) DeleteKey(id int64) error {
	_, err := m.db.Exec(`DELETE FROM api_keys WHERE id=?`, id)
	return err
}

// IsModelAllowed checks if a model is in the allowed list
func (ak *APIKey) IsModelAllowed(model string) bool {
	if ak.AllowedModels == "" {
		return true // empty means all models allowed
	}
	for _, m := range strings.Split(ak.AllowedModels, ",") {
		if strings.TrimSpace(m) == model {
			return true
		}
	}
	return false
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
