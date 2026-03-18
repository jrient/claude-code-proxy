package provider

import (
	"bytes"
	"fmt"
	"net/http"
)

// AnthropicBackend handles native Anthropic API forwarding
type AnthropicBackend struct{}

func NewAnthropicBackend() *AnthropicBackend {
	return &AnthropicBackend{}
}

func (a *AnthropicBackend) ProviderType() string {
	return "anthropic"
}

func (a *AnthropicBackend) TransformRequest(body []byte, model string) ([]byte, string, error) {
	// For Anthropic native, we pass through the body as-is
	// Only replace the model if a mapping is specified
	return body, model, nil
}

func (a *AnthropicBackend) Send(req *http.Request, p *Provider, model string) (*http.Response, error) {
	body, _, err := a.TransformRequest(nil, model)
	_ = body

	// Build the upstream request
	upstreamURL := fmt.Sprintf("%s/v1/messages", p.BaseURL)

	upstreamReq, err := http.NewRequestWithContext(req.Context(), req.Method, upstreamURL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	// Copy relevant headers
	for _, h := range []string{"Content-Type", "Accept", "Anthropic-Version", "Anthropic-Beta"} {
		if v := req.Header.Get(h); v != "" {
			upstreamReq.Header.Set(h, v)
		}
	}

	// Set the provider's API key
	upstreamReq.Header.Set("X-Api-Key", p.APIKey)
	upstreamReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(upstreamReq)
}

// SendWithBody sends a request with a pre-built body
func (a *AnthropicBackend) SendWithBody(req *http.Request, p *Provider, body []byte) (*http.Response, error) {
	upstreamURL := fmt.Sprintf("%s/v1/messages", p.BaseURL)

	upstreamReq, err := http.NewRequestWithContext(req.Context(), req.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	for _, h := range []string{"Content-Type", "Accept", "Anthropic-Version", "Anthropic-Beta"} {
		if v := req.Header.Get(h); v != "" {
			upstreamReq.Header.Set(h, v)
		}
	}

	upstreamReq.Header.Set("X-Api-Key", p.APIKey)
	upstreamReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(upstreamReq)
}
