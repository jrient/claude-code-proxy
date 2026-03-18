package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// OpenAIBackend handles OpenAI-compatible API providers (OpenRouter, etc.)
type OpenAIBackend struct{}

func NewOpenAIBackend() *OpenAIBackend {
	return &OpenAIBackend{}
}

func (o *OpenAIBackend) ProviderType() string {
	return "openai"
}

// TransformRequest converts Anthropic Messages API format to OpenAI Chat Completions format
func (o *OpenAIBackend) TransformRequest(body []byte, targetModel string) ([]byte, string, error) {
	var anthropicReq map[string]interface{}
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		return nil, "", fmt.Errorf("parse request: %w", err)
	}

	openaiReq := map[string]interface{}{
		"model": targetModel,
	}

	// Convert messages
	var openaiMessages []map[string]interface{}

	// Handle system message
	if sys, ok := anthropicReq["system"]; ok && sys != nil {
		sysText := extractSystemText(sys)
		if sysText != "" {
			openaiMessages = append(openaiMessages, map[string]interface{}{
				"role":    "system",
				"content": sysText,
			})
		}
	}

	// Convert messages array
	if msgs, ok := anthropicReq["messages"].([]interface{}); ok {
		for _, m := range msgs {
			msg, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			openaiMsg := map[string]interface{}{
				"role":    msg["role"],
				"content": convertContent(msg["content"]),
			}
			openaiMessages = append(openaiMessages, openaiMsg)
		}
	}

	openaiReq["messages"] = openaiMessages

	// Map parameters
	if maxTokens, ok := anthropicReq["max_tokens"]; ok {
		openaiReq["max_tokens"] = maxTokens
	}
	if temp, ok := anthropicReq["temperature"]; ok {
		openaiReq["temperature"] = temp
	}
	if topP, ok := anthropicReq["top_p"]; ok {
		openaiReq["top_p"] = topP
	}
	if stream, ok := anthropicReq["stream"].(bool); ok && stream {
		openaiReq["stream"] = true
	}

	result, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, "", fmt.Errorf("marshal openai request: %w", err)
	}

	return result, targetModel, nil
}

func extractSystemText(sys interface{}) string {
	switch v := sys.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, block := range v {
			if m, ok := block.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		result := ""
		for i, p := range parts {
			if i > 0 {
				result += "\n"
			}
			result += p
		}
		return result
	}
	return ""
}

func convertContent(content interface{}) interface{} {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		// Convert content blocks to text
		var text string
		for _, block := range v {
			if m, ok := block.(map[string]interface{}); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if txt, ok := m["text"].(string); ok {
						text += txt
					}
				}
			}
		}
		if text != "" {
			return text
		}
		return v
	}
	return content
}

func (o *OpenAIBackend) Send(req *http.Request, p *Provider, model string) (*http.Response, error) {
	return nil, fmt.Errorf("use SendWithBody for OpenAI backend")
}

// SendWithBody sends the transformed request to the OpenAI-compatible endpoint
func (o *OpenAIBackend) SendWithBody(req *http.Request, p *Provider, body []byte) (*http.Response, error) {
	upstreamURL := fmt.Sprintf("%s/chat/completions", p.BaseURL)

	upstreamReq, err := http.NewRequestWithContext(req.Context(), "POST", upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	// Copy Accept header for streaming
	if v := req.Header.Get("Accept"); v != "" {
		upstreamReq.Header.Set("Accept", v)
	}

	client := &http.Client{}
	return client.Do(upstreamReq)
}
