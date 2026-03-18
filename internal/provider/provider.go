package provider

import (
	"io"
	"net/http"
)

// Provider represents a backend API provider
type Provider struct {
	ID           int64
	Name         string
	Type         string // "anthropic" or "openai"
	BaseURL      string
	APIKey       string
	Priority     int
	Weight       int
	Enabled      bool
	HealthStatus string // "healthy", "unhealthy", "unknown"

	// Runtime state
	ConsecutiveFails int
	CircuitOpen      bool
}

// Request represents a normalized proxy request
type Request struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Stream      bool
	Temperature *float64
	System      interface{} // string or []SystemBlock
	RawBody     []byte
	Headers     http.Header
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type SystemBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Response represents the proxy response to send back
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
	Stream     bool
}

// Backend is the interface for different provider implementations
type Backend interface {
	// Send sends a request and returns a response (handles both stream and non-stream)
	Send(req *http.Request, providerCfg *Provider, model string) (*http.Response, error)
	// TransformRequest transforms the incoming Anthropic-format request for this provider
	TransformRequest(body []byte, model string) ([]byte, string, error)
	// ProviderType returns the type name
	ProviderType() string
}
