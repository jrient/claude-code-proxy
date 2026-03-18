package admin

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user/claude-code-proxy/internal/auth"
	"github.com/user/claude-code-proxy/internal/provider"
	"github.com/user/claude-code-proxy/internal/router"
	"github.com/user/claude-code-proxy/internal/stats"
)

type API struct {
	db         *sql.DB
	registry   *provider.Registry
	router     *router.Router
	keyManager *auth.APIKeyManager
	aggregator *stats.Aggregator
	adminPass  string
}

func NewAPI(db *sql.DB, registry *provider.Registry, r *router.Router, keyManager *auth.APIKeyManager, aggregator *stats.Aggregator, adminPass string) *API {
	return &API{
		db:         db,
		registry:   registry,
		router:     r,
		keyManager: keyManager,
		aggregator: aggregator,
		adminPass:  adminPass,
	}
}

func (a *API) RegisterRoutes(r *gin.Engine) {
	// Login
	r.POST("/api/login", a.handleLogin)

	// Protected routes
	api := r.Group("/api")
	api.Use(auth.AdminAuthMiddleware(a.adminPass))
	{
		// Dashboard
		api.GET("/dashboard", a.handleDashboard)

		// Providers
		api.GET("/providers", a.handleListProviders)
		api.POST("/providers", a.handleCreateProvider)
		api.PUT("/providers/:id", a.handleUpdateProvider)
		api.DELETE("/providers/:id", a.handleDeleteProvider)

		// API Keys
		api.GET("/apikeys", a.handleListAPIKeys)
		api.POST("/apikeys", a.handleCreateAPIKey)
		api.PUT("/apikeys/:id", a.handleUpdateAPIKey)
		api.DELETE("/apikeys/:id", a.handleDeleteAPIKey)

		// Model Mappings
		api.GET("/providers/:id/models", a.handleListModelMappings)
		api.POST("/providers/:id/models", a.handleCreateModelMapping)
		api.DELETE("/providers/:id/models/:mapping_id", a.handleDeleteModelMapping)

		// Stats
		api.GET("/stats/timeseries", a.handleTimeSeries)
		api.GET("/stats/models", a.handleModelStats)
		api.GET("/stats/logs", a.handleRecentLogs)
	}
}

func (a *API) handleLogin(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Password != a.adminPass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": a.adminPass})
}

func (a *API) handleDashboard(c *gin.Context) {
	dashStats, err := a.aggregator.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add active providers count
	providers := a.registry.GetEnabledProviders()
	dashStats.ActiveProviders = len(providers)

	c.JSON(http.StatusOK, dashStats)
}

// --- Provider CRUD ---

func (a *API) handleListProviders(c *gin.Context) {
	rows, err := a.db.Query(`SELECT id, name, type, base_url, api_key, priority, weight, enabled, health_status, config_json, created_at, updated_at FROM providers ORDER BY priority`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var providers []map[string]interface{}
	for rows.Next() {
		var id, priority, weight int64
		var enabled int
		var name, pType, baseURL, apiKey, healthStatus, configJSON, createdAt, updatedAt string
		rows.Scan(&id, &name, &pType, &baseURL, &apiKey, &priority, &weight, &enabled, &healthStatus, &configJSON, &createdAt, &updatedAt)

		// Mask API key
		maskedKey := apiKey
		if len(apiKey) > 8 {
			maskedKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		}

		providers = append(providers, map[string]interface{}{
			"id":            id,
			"name":          name,
			"type":          pType,
			"base_url":      baseURL,
			"api_key":       maskedKey,
			"priority":      priority,
			"weight":        weight,
			"enabled":       enabled == 1,
			"health_status": healthStatus,
			"config_json":   configJSON,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		})
	}
	if providers == nil {
		providers = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, providers)
}

func (a *API) handleCreateProvider(c *gin.Context) {
	var req struct {
		Name       string                   `json:"name"`
		Type       string                   `json:"type"`
		BaseURL    string                   `json:"base_url"`
		APIKey     string                   `json:"api_key"`
		Priority   int                      `json:"priority"`
		Weight     int                      `json:"weight"`
		Models     []map[string]string      `json:"models"`
		ConfigJSON string                   `json:"config_json"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Weight == 0 {
		req.Weight = 10
	}
	if req.Priority == 0 {
		req.Priority = 1
	}
	if req.ConfigJSON == "" {
		req.ConfigJSON = "{}"
	}

	result, err := a.db.Exec(
		`INSERT INTO providers (name, type, base_url, api_key, priority, weight, config_json) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Type, req.BaseURL, req.APIKey, req.Priority, req.Weight, req.ConfigJSON,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()

	// Add to registry
	p := &provider.Provider{
		ID:           id,
		Name:         req.Name,
		Type:         req.Type,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Priority:     req.Priority,
		Weight:       req.Weight,
		Enabled:      true,
		HealthStatus: "unknown",
	}
	a.registry.AddProvider(p)

	// Set model mappings
	for _, m := range req.Models {
		if src, ok := m["source"]; ok {
			if tgt, ok := m["target"]; ok {
				a.router.SetModelMapping(id, src, tgt)
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (a *API) handleUpdateProvider(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
		Priority int    `json:"priority"`
		Weight   int    `json:"weight"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}

	_, err := a.db.Exec(
		`UPDATE providers SET name=?, type=?, base_url=?, api_key=?, priority=?, weight=?, enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		req.Name, req.Type, req.BaseURL, req.APIKey, req.Priority, req.Weight, enabled, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update registry
	p := a.registry.GetProvider(id)
	if p != nil {
		p.Name = req.Name
		p.Type = req.Type
		p.BaseURL = req.BaseURL
		p.APIKey = req.APIKey
		p.Priority = req.Priority
		p.Weight = req.Weight
		p.Enabled = enabled == 1
		a.registry.UpdateProvider(p)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) handleDeleteProvider(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	a.db.Exec(`DELETE FROM providers WHERE id=?`, id)
	a.registry.RemoveProvider(id)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- API Keys ---

func (a *API) handleListAPIKeys(c *gin.Context) {
	keys, err := a.keyManager.ListKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if keys == nil {
		keys = []auth.APIKey{}
	}
	c.JSON(http.StatusOK, keys)
}

func (a *API) handleCreateAPIKey(c *gin.Context) {
	var req struct {
		Name            string `json:"name"`
		RateLimit       int    `json:"rate_limit"`
		DailyTokenLimit int    `json:"daily_token_limit"`
		AllowedModels   string `json:"allowed_models"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RateLimit == 0 {
		req.RateLimit = 60
	}

	fullKey, key, err := a.keyManager.GenerateKey(req.Name, req.RateLimit, req.DailyTokenLimit, req.AllowedModels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":  fullKey,
		"info": key,
	})
}

func (a *API) handleUpdateAPIKey(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Name            string `json:"name"`
		Enabled         bool   `json:"enabled"`
		RateLimit       int    `json:"rate_limit"`
		DailyTokenLimit int    `json:"daily_token_limit"`
		AllowedModels   string `json:"allowed_models"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := a.keyManager.UpdateKey(id, req.Name, req.Enabled, req.RateLimit, req.DailyTokenLimit, req.AllowedModels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) handleDeleteAPIKey(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := a.keyManager.DeleteKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Model Mappings ---

func (a *API) handleListModelMappings(c *gin.Context) {
	providerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	rows, err := a.db.Query(`SELECT id, source_model, target_model FROM model_mappings WHERE provider_id = ? ORDER BY id`, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var mappings []map[string]interface{}
	for rows.Next() {
		var id int64
		var source, target string
		rows.Scan(&id, &source, &target)
		mappings = append(mappings, map[string]interface{}{
			"id":     id,
			"source": source,
			"target": target,
		})
	}
	if mappings == nil {
		mappings = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, mappings)
}

func (a *API) handleCreateModelMapping(c *gin.Context) {
	providerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Source == "" || req.Target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source and target are required"})
		return
	}

	result, err := a.db.Exec(
		`INSERT INTO model_mappings (provider_id, source_model, target_model) VALUES (?, ?, ?)
		 ON CONFLICT(provider_id, source_model) DO UPDATE SET target_model=excluded.target_model`,
		providerID, req.Source, req.Target,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	// Update in-memory router mapping
	a.router.SetModelMapping(providerID, req.Source, req.Target)

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (a *API) handleDeleteModelMapping(c *gin.Context) {
	providerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	mappingID, _ := strconv.ParseInt(c.Param("mapping_id"), 10, 64)

	// Get source_model before deleting so we can remove from router
	var source string
	err := a.db.QueryRow(`SELECT source_model FROM model_mappings WHERE id = ? AND provider_id = ?`, mappingID, providerID).Scan(&source)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mapping not found"})
		return
	}

	_, err = a.db.Exec(`DELETE FROM model_mappings WHERE id = ? AND provider_id = ?`, mappingID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove from in-memory router mapping
	a.router.RemoveModelMapping(providerID, source)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Stats ---

func (a *API) handleTimeSeries(c *gin.Context) {
	period := c.DefaultQuery("period", "hour")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	apiKeyID, _ := strconv.ParseInt(c.DefaultQuery("api_key_id", "0"), 10, 64)
	providerID, _ := strconv.ParseInt(c.DefaultQuery("provider_id", "0"), 10, 64)

	points, err := a.aggregator.GetTimeSeries(period, days, apiKeyID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if points == nil {
		points = []stats.TimeSeriesPoint{}
	}
	c.JSON(http.StatusOK, points)
}

func (a *API) handleModelStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	results, err := a.aggregator.GetModelStats(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, results)
}

func (a *API) handleRecentLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, total, err := a.aggregator.GetRecentLogs(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []stats.RequestLog{}
	}
	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
	})
}
