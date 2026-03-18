package router

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/user/claude-code-proxy/internal/provider"
)

// Router handles provider selection and load balancing
type Router struct {
	registry *provider.Registry
	mu       sync.Mutex
	rrIndex  atomic.Int64 // weighted round-robin index

	// Model mappings: provider_id -> (source_model -> target_model)
	modelMappings sync.Map
}

func New(registry *provider.Registry) *Router {
	return &Router{
		registry: registry,
	}
}

// SetModelMapping sets a model mapping for a provider
func (r *Router) SetModelMapping(providerID int64, source, target string) {
	key := providerID
	val, _ := r.modelMappings.LoadOrStore(key, &sync.Map{})
	val.(*sync.Map).Store(source, target)
}

// RemoveModelMapping removes a model mapping for a provider
func (r *Router) RemoveModelMapping(providerID int64, source string) {
	val, ok := r.modelMappings.Load(providerID)
	if !ok {
		return
	}
	val.(*sync.Map).Delete(source)
}

// GetTargetModel returns the mapped model name for a provider, or the original if no mapping
func (r *Router) GetTargetModel(providerID int64, sourceModel string) string {
	val, ok := r.modelMappings.Load(providerID)
	if !ok {
		return sourceModel
	}
	if target, ok := val.(*sync.Map).Load(sourceModel); ok {
		return target.(string)
	}
	return sourceModel
}

// SelectProvider picks the best available provider using priority + weighted round-robin
func (r *Router) SelectProvider(model string) *provider.Provider {
	providers := r.registry.GetEnabledProviders()
	if len(providers) == 0 {
		return nil
	}

	// Sort by priority (lower = higher priority)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority < providers[j].Priority
	})

	// Group by priority - pick from highest priority group
	highestPriority := providers[0].Priority
	var candidates []*provider.Provider
	for _, p := range providers {
		if p.Priority == highestPriority {
			// Check if this provider has a mapping for the requested model
			// or is an anthropic provider (passes through)
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	if len(candidates) == 1 {
		return candidates[0]
	}

	// Weighted round-robin among same-priority providers
	return r.weightedSelect(candidates)
}

// SelectProviderWithFallback tries each provider in priority order
func (r *Router) SelectProviderWithFallback(model string, excludeIDs []int64) *provider.Provider {
	providers := r.registry.GetEnabledProviders()
	if len(providers) == 0 {
		return nil
	}

	excludeSet := make(map[int64]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	// Sort by priority
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority < providers[j].Priority
	})

	for _, p := range providers {
		if !excludeSet[p.ID] {
			return p
		}
	}

	return nil
}

func (r *Router) weightedSelect(candidates []*provider.Provider) *provider.Provider {
	totalWeight := 0
	for _, p := range candidates {
		totalWeight += p.Weight
	}

	if totalWeight == 0 {
		idx := r.rrIndex.Add(1) % int64(len(candidates))
		return candidates[idx]
	}

	counter := int(r.rrIndex.Add(1)) % totalWeight
	for _, p := range candidates {
		counter -= p.Weight
		if counter < 0 {
			return p
		}
	}

	return candidates[0]
}

// MarkFailed records a failure for a provider and opens circuit if threshold exceeded
func (r *Router) MarkFailed(p *provider.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ConsecutiveFails++
	if p.ConsecutiveFails >= 3 {
		p.CircuitOpen = true
		p.HealthStatus = "unhealthy"
	}
}

// MarkSuccess resets failure counter for a provider
func (r *Router) MarkSuccess(p *provider.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ConsecutiveFails = 0
	p.CircuitOpen = false
	p.HealthStatus = "healthy"
}
