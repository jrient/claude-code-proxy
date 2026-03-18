package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/user/claude-code-proxy/internal/admin"
	"github.com/user/claude-code-proxy/internal/auth"
	"github.com/user/claude-code-proxy/internal/config"
	"github.com/user/claude-code-proxy/internal/db"
	"github.com/user/claude-code-proxy/internal/provider"
	"github.com/user/claude-code-proxy/internal/proxy"
	"github.com/user/claude-code-proxy/internal/router"
	"github.com/user/claude-code-proxy/internal/stats"
)

//go:embed all:dist
var frontendFS embed.FS

func main() {
	// Load config
	cfgPath := "config.yaml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		cfgPath = v
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("[main] Warning: config file not found (%s), using defaults", cfgPath)
		cfg = &config.Config{
			Server:   config.ServerConfig{Port: 8080, AdminPort: 8081},
			Auth:     config.AuthConfig{AdminPassword: "changeme"},
			Database: config.DatabaseConfig{Path: "./data/proxy.db"},
		}
	}
	cfg.Validate()

	log.Printf("[main] Starting Claude Code Proxy")
	log.Printf("[main] Proxy port: %d, Admin port: %d", cfg.Server.Port, cfg.Server.AdminPort)

	// Initialize database
	database, err := db.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("[main] Database init failed: %v", err)
	}
	defer database.Close()

	// Initialize components
	registry := provider.NewRegistry()
	rt := router.New(registry)
	keyManager := auth.NewAPIKeyManager(database.DB)
	rateLimiter := auth.NewRateLimiter()
	collector := stats.NewCollector(database.DB)
	aggregator := stats.NewAggregator(database.DB)
	proxyHandler := proxy.NewHandler(registry, rt, collector)

	// Load providers from config
	for _, pc := range cfg.Providers {
		apiKey := config.ExpandEnvInString(pc.APIKey)
		result, err := database.Exec(
			`INSERT OR IGNORE INTO providers (name, type, base_url, api_key, priority, weight) VALUES (?, ?, ?, ?, ?, ?)`,
			pc.Name, pc.Type, pc.BaseURL, apiKey, pc.Priority, pc.Weight,
		)
		if err != nil {
			log.Printf("[main] Failed to insert provider %s: %v", pc.Name, err)
			continue
		}

		id, _ := result.LastInsertId()
		if id == 0 {
			// Already exists, get its ID
			database.QueryRow(`SELECT id FROM providers WHERE name = ?`, pc.Name).Scan(&id)
		}

		p := &provider.Provider{
			ID:           id,
			Name:         pc.Name,
			Type:         pc.Type,
			BaseURL:      pc.BaseURL,
			APIKey:       apiKey,
			Priority:     pc.Priority,
			Weight:       pc.Weight,
			Enabled:      true,
			HealthStatus: "unknown",
		}
		registry.AddProvider(p)

		// Set model mappings
		for _, m := range pc.Models {
			rt.SetModelMapping(id, m.Source, m.Target)
		}

		log.Printf("[main] Loaded provider: %s (%s) priority=%d", pc.Name, pc.Type, pc.Priority)
	}

	// Also load providers from database that aren't in config
	rows, err := database.Query(`SELECT id, name, type, base_url, api_key, priority, weight, enabled, health_status FROM providers`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p provider.Provider
			var enabled int
			rows.Scan(&p.ID, &p.Name, &p.Type, &p.BaseURL, &p.APIKey, &p.Priority, &p.Weight, &enabled, &p.HealthStatus)
			p.Enabled = enabled == 1
			// Only add if not already in registry
			if existing := registry.GetProvider(p.ID); existing == nil {
				registry.AddProvider(&p)
			}
		}
	}

	// Start stats collector
	collector.Start()
	defer collector.Stop()

	// Start health checker
	healthChecker := router.NewHealthChecker(registry, rt, 30*time.Second)
	healthChecker.Start()
	defer healthChecker.Stop()

	// --- Proxy Server ---
	gin.SetMode(gin.ReleaseMode)

	proxyEngine := gin.New()
	proxyEngine.Use(gin.Recovery())
	proxyEngine.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"*"},
	}))

	// Proxy routes
	proxyEngine.POST("/v1/messages", auth.AuthMiddleware(keyManager, rateLimiter), proxyHandler.HandleMessages)

	// Health check
	proxyEngine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// --- Admin Server ---
	adminEngine := gin.New()
	adminEngine.Use(gin.Recovery())
	adminEngine.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"*"},
	}))

	// Admin API
	adminAPI := admin.NewAPI(database.DB, registry, rt, keyManager, aggregator, cfg.Auth.AdminPassword)
	adminAPI.RegisterRoutes(adminEngine)

	// Serve frontend static files
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Printf("[main] Warning: frontend assets not found: %v", err)
	} else {
		adminEngine.NoRoute(func(c *gin.Context) {
			// Try to serve the file
			path := c.Request.URL.Path
			f, err := distFS.Open(path[1:]) // Remove leading /
			if err == nil {
				f.Close()
				c.FileFromFS(path, http.FS(distFS))
				return
			}
			// Fallback to index.html for SPA routing
			c.FileFromFS("/", http.FS(distFS))
		})
	}

	// Start servers
	proxySrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: proxyEngine,
	}
	adminSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.AdminPort),
		Handler: adminEngine,
	}

	go func() {
		log.Printf("[main] Proxy server listening on :%d", cfg.Server.Port)
		if err := proxySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[proxy] server error: %v", err)
		}
	}()

	go func() {
		log.Printf("[main] Admin server listening on :%d", cfg.Server.AdminPort)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[admin] server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[main] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proxySrv.Shutdown(ctx)
	adminSrv.Shutdown(ctx)
	log.Println("[main] Bye!")
}
