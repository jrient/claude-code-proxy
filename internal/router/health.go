package router

import (
	"log"
	"net/http"
	"time"

	"github.com/user/claude-code-proxy/internal/provider"
)

// HealthChecker periodically checks provider health
type HealthChecker struct {
	registry *provider.Registry
	router   *Router
	interval time.Duration
	stop     chan struct{}
}

func NewHealthChecker(registry *provider.Registry, router *Router, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		router:   router,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (h *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.checkAll()
			case <-h.stop:
				return
			}
		}
	}()
}

func (h *HealthChecker) Stop() {
	close(h.stop)
}

func (h *HealthChecker) checkAll() {
	providers := h.registry.GetProviders()
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		go h.checkOne(p)
	}
}

func (h *HealthChecker) checkOne(p *provider.Provider) {
	healthy := false

	switch p.Type {
	case "anthropic":
		healthy = h.checkAnthropic(p)
	case "openai":
		healthy = h.checkOpenAI(p)
	default:
		healthy = h.checkGeneric(p)
	}

	if healthy {
		h.router.MarkSuccess(p)
	} else {
		log.Printf("[health] provider %s (%s) is unhealthy", p.Name, p.BaseURL)
	}
}

func (h *HealthChecker) checkAnthropic(p *provider.Provider) bool {
	// Simple connectivity check - just check if the base URL is reachable
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(p.BaseURL)
	if err != nil {
		return false
	}
	resp.Body.Close()
	// Any response (even 404) means the server is up
	return true
}

func (h *HealthChecker) checkOpenAI(p *provider.Provider) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", p.BaseURL+"/models", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

func (h *HealthChecker) checkGeneric(p *provider.Provider) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(p.BaseURL)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
