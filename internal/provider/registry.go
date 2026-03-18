package provider

import (
	"fmt"
	"sync"
)

// Registry manages all configured providers
type Registry struct {
	mu        sync.RWMutex
	providers []*Provider
	backends  map[string]Backend // type -> backend implementation
}

func NewRegistry() *Registry {
	r := &Registry{
		providers: make([]*Provider, 0),
		backends:  make(map[string]Backend),
	}
	r.backends["anthropic"] = NewAnthropicBackend()
	r.backends["openai"] = NewOpenAIBackend()
	return r
}

func (r *Registry) AddProvider(p *Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, p)
}

func (r *Registry) RemoveProvider(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.providers {
		if p.ID == id {
			r.providers = append(r.providers[:i], r.providers[i+1:]...)
			return
		}
	}
}

func (r *Registry) GetProvider(id int64) *Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (r *Registry) GetProviders() []*Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Provider, len(r.providers))
	copy(result, r.providers)
	return result
}

func (r *Registry) GetEnabledProviders() []*Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Provider
	for _, p := range r.providers {
		if p.Enabled && !p.CircuitOpen {
			result = append(result, p)
		}
	}
	return result
}

func (r *Registry) GetBackend(providerType string) (Backend, error) {
	b, ok := r.backends[providerType]
	if !ok {
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
	return b, nil
}

func (r *Registry) UpdateProvider(p *Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.providers {
		if existing.ID == p.ID {
			r.providers[i] = p
			return
		}
	}
}
