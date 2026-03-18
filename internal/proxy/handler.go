package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user/claude-code-proxy/internal/auth"
	"github.com/user/claude-code-proxy/internal/provider"
	"github.com/user/claude-code-proxy/internal/router"
	"github.com/user/claude-code-proxy/internal/stats"
)

type Handler struct {
	registry  *provider.Registry
	router    *router.Router
	collector *stats.Collector
}

func NewHandler(registry *provider.Registry, r *router.Router, collector *stats.Collector) *Handler {
	return &Handler{
		registry:  registry,
		router:    r,
		collector: collector,
	}
}

// HandleMessages handles POST /v1/messages - the main Anthropic Messages API endpoint
func (h *Handler) HandleMessages(c *gin.Context) {
	startTime := time.Now()

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{"type": "invalid_request_error", "message": "Failed to read request body"},
		})
		return
	}

	// Parse the request to get model and stream info
	var reqBody struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{"type": "invalid_request_error", "message": "Invalid JSON body"},
		})
		return
	}

	// Check model access
	if apiKey, exists := c.Get("api_key"); exists {
		ak := apiKey.(*auth.APIKey)
		if !ak.IsModelAllowed(reqBody.Model) {
			c.JSON(http.StatusForbidden, gin.H{
				"type":  "error",
				"error": gin.H{"type": "permission_error", "message": "Model not allowed for this API key"},
			})
			return
		}
	}

	// Try providers with fallback
	var excludeIDs []int64
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		var p *provider.Provider
		if attempt == 0 {
			p = h.router.SelectProvider(reqBody.Model)
		} else {
			p = h.router.SelectProviderWithFallback(reqBody.Model, excludeIDs)
		}

		if p == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"type":  "error",
				"error": gin.H{"type": "api_error", "message": "No available providers"},
			})
			return
		}

		resp, err := h.sendToProvider(c.Request, p, body, reqBody.Model)
		if err != nil {
			log.Printf("[proxy] provider %s error: %v", p.Name, err)
			h.router.MarkFailed(p)
			excludeIDs = append(excludeIDs, p.ID)
			continue
		}

		// Check if response indicates provider error (5xx)
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			h.router.MarkFailed(p)
			excludeIDs = append(excludeIDs, p.ID)
			continue
		}

		h.router.MarkSuccess(p)

		// Record stats
		apiKeyID := int64(0)
		if id, exists := c.Get("api_key_id"); exists {
			apiKeyID = id.(int64)
		}

		if reqBody.Stream {
			h.handleStreamResponse(c, resp, p, apiKeyID, reqBody.Model, startTime)
		} else {
			h.handleNormalResponse(c, resp, p, apiKeyID, reqBody.Model, startTime)
		}
		return
	}

	c.JSON(http.StatusBadGateway, gin.H{
		"type":  "error",
		"error": gin.H{"type": "api_error", "message": "All providers failed"},
	})
}

func (h *Handler) sendToProvider(req *http.Request, p *provider.Provider, body []byte, model string) (*http.Response, error) {
	targetModel := h.router.GetTargetModel(p.ID, model)

	switch p.Type {
	case "openai":
		backend := provider.NewOpenAIBackend()
		transformedBody, _, err := backend.TransformRequest(body, targetModel)
		if err != nil {
			return nil, err
		}
		return backend.SendWithBody(req, p, transformedBody)
	default: // anthropic
		// Replace model in body if mapped
		if targetModel != model {
			var bodyMap map[string]interface{}
			json.Unmarshal(body, &bodyMap)
			bodyMap["model"] = targetModel
			body, _ = json.Marshal(bodyMap)
		}
		backend := provider.NewAnthropicBackend()
		return backend.SendWithBody(req, p, body)
	}
}

func (h *Handler) handleNormalResponse(c *gin.Context, resp *http.Response, p *provider.Provider, apiKeyID int64, model string, startTime time.Time) {
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"type":  "error",
			"error": gin.H{"type": "api_error", "message": "Failed to read upstream response"},
		})
		return
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// For OpenAI providers, transform response back to Anthropic format
	if p.Type == "openai" && resp.StatusCode == 200 {
		anthropicResp, promptTokens, completionTokens, err := TransformOpenAIToAnthropic(respBody, model)
		if err == nil {
			respBody = anthropicResp
			// Record stats
			h.collector.Record(&stats.RequestLog{
				APIKeyID:         apiKeyID,
				ProviderID:       p.ID,
				Model:            model,
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      promptTokens + completionTokens,
				LatencyMs:        latencyMs,
				StatusCode:       resp.StatusCode,
				Stream:           false,
			})
		} else {
			log.Printf("[proxy] transform response error: %v", err)
		}
	} else {
		// Extract tokens from Anthropic response
		var anthropicResp struct {
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(respBody, &anthropicResp) == nil {
			h.collector.Record(&stats.RequestLog{
				APIKeyID:         apiKeyID,
				ProviderID:       p.ID,
				Model:            model,
				PromptTokens:     anthropicResp.Usage.InputTokens,
				CompletionTokens: anthropicResp.Usage.OutputTokens,
				TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
				LatencyMs:        latencyMs,
				StatusCode:       resp.StatusCode,
				Stream:           false,
			})
		}
	}

	// Copy response headers
	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Header(k, v)
		}
	}
	c.Data(resp.StatusCode, "application/json", respBody)
}

func (h *Handler) handleStreamResponse(c *gin.Context, resp *http.Response, p *provider.Provider, apiKeyID int64, model string, startTime time.Time) {
	defer resp.Body.Close()

	if p.Type == "openai" {
		h.handleOpenAIStreamResponse(c, resp, p, apiKeyID, model, startTime)
		return
	}

	// For Anthropic native, just pass through the SSE stream
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(resp.StatusCode)

	var totalPrompt, totalCompletion int

	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// Try to extract token usage from SSE events
			data := buf[:n]
			prompt, completion := extractAnthropicStreamTokens(data)
			if prompt > 0 {
				totalPrompt = prompt
			}
			if completion > 0 {
				totalCompletion = completion
			}

			c.Writer.Write(data)
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			break
		}
	}

	latencyMs := time.Since(startTime).Milliseconds()
	h.collector.Record(&stats.RequestLog{
		APIKeyID:         apiKeyID,
		ProviderID:       p.ID,
		Model:            model,
		PromptTokens:     totalPrompt,
		CompletionTokens: totalCompletion,
		TotalTokens:      totalPrompt + totalCompletion,
		LatencyMs:        latencyMs,
		StatusCode:       resp.StatusCode,
		Stream:           true,
	})
}

func (h *Handler) handleOpenAIStreamResponse(c *gin.Context, resp *http.Response, p *provider.Provider, apiKeyID int64, model string, startTime time.Time) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(200)

	flusher, _ := c.Writer.(http.Flusher)

	streamer := NewOpenAIStreamTransformer(model)
	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			events := streamer.Transform(buf[:n])
			for _, event := range events {
				c.Writer.Write(event)
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
		if err != nil {
			// Send final events
			finalEvents := streamer.Finalize()
			for _, event := range finalEvents {
				c.Writer.Write(event)
				if flusher != nil {
					flusher.Flush()
				}
			}
			break
		}
	}

	latencyMs := time.Since(startTime).Milliseconds()
	h.collector.Record(&stats.RequestLog{
		APIKeyID:         apiKeyID,
		ProviderID:       p.ID,
		Model:            model,
		PromptTokens:     streamer.PromptTokens,
		CompletionTokens: streamer.CompletionTokens,
		TotalTokens:      streamer.PromptTokens + streamer.CompletionTokens,
		LatencyMs:        latencyMs,
		StatusCode:       resp.StatusCode,
		Stream:           true,
	})
}

func extractAnthropicStreamTokens(data []byte) (prompt, completion int) {
	// Look for message_delta events with usage info
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		payload := bytes.TrimPrefix(line, []byte("data: "))
		var event struct {
			Type  string `json:"type"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Message struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(payload, &event) == nil {
			if event.Message.Usage.InputTokens > 0 {
				prompt = event.Message.Usage.InputTokens
			}
			if event.Usage.OutputTokens > 0 {
				completion = event.Usage.OutputTokens
			}
		}
	}
	return
}
