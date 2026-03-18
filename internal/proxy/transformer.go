package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TransformOpenAIToAnthropic converts an OpenAI chat completion response to Anthropic Messages format
func TransformOpenAIToAnthropic(body []byte, model string) ([]byte, int, int, error) {
	var openaiResp struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, 0, 0, fmt.Errorf("parse openai response: %w", err)
	}

	// Map finish reason
	stopReason := "end_turn"
	if len(openaiResp.Choices) > 0 {
		switch openaiResp.Choices[0].FinishReason {
		case "length":
			stopReason = "max_tokens"
		case "stop":
			stopReason = "end_turn"
		}
	}

	content := ""
	if len(openaiResp.Choices) > 0 {
		content = openaiResp.Choices[0].Message.Content
	}

	anthropicResp := map[string]interface{}{
		"id":   "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:24],
		"type": "message",
		"role": "assistant",
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
		"model":       model,
		"stop_reason": stopReason,
		"usage": map[string]interface{}{
			"input_tokens":  openaiResp.Usage.PromptTokens,
			"output_tokens": openaiResp.Usage.CompletionTokens,
		},
	}

	result, err := json.Marshal(anthropicResp)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("marshal anthropic response: %w", err)
	}

	return result, openaiResp.Usage.PromptTokens, openaiResp.Usage.CompletionTokens, nil
}

// OpenAIStreamTransformer converts OpenAI SSE stream to Anthropic SSE stream
type OpenAIStreamTransformer struct {
	model            string
	msgID            string
	buffer           []byte
	contentSoFar     string
	PromptTokens     int
	CompletionTokens int
	started          bool
}

func NewOpenAIStreamTransformer(model string) *OpenAIStreamTransformer {
	return &OpenAIStreamTransformer{
		model: model,
		msgID: "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:24],
	}
}

func (t *OpenAIStreamTransformer) Transform(data []byte) [][]byte {
	t.buffer = append(t.buffer, data...)
	var events [][]byte

	for {
		idx := bytes.Index(t.buffer, []byte("\n\n"))
		if idx == -1 {
			break
		}

		line := t.buffer[:idx]
		t.buffer = t.buffer[idx+2:]

		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		payload := bytes.TrimPrefix(line, []byte("data: "))
		if string(payload) == "[DONE]" {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
					Role    string `json:"role"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal(payload, &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			t.PromptTokens = chunk.Usage.PromptTokens
			t.CompletionTokens = chunk.Usage.CompletionTokens
		}

		if !t.started {
			// Send message_start event
			t.started = true
			msgStart := fmt.Sprintf(`event: message_start
data: {"type":"message_start","message":{"id":"%s","type":"message","role":"assistant","content":[],"model":"%s","stop_reason":null,"usage":{"input_tokens":0,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: ping
data: {"type":"ping"}

`, t.msgID, t.model)
			events = append(events, []byte(msgStart))
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				t.contentSoFar += delta
				// Send content_block_delta
				deltaJSON, _ := json.Marshal(delta)
				event := fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%s}}\n\n", deltaJSON)
				events = append(events, []byte(event))
			}

			if chunk.Choices[0].FinishReason != nil {
				stopReason := "end_turn"
				if *chunk.Choices[0].FinishReason == "length" {
					stopReason = "max_tokens"
				}

				final := fmt.Sprintf(`event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"%s"},"usage":{"output_tokens":%d}}

event: message_stop
data: {"type":"message_stop"}

`, stopReason, t.CompletionTokens)
				events = append(events, []byte(final))
			}
		}
	}

	return events
}

func (t *OpenAIStreamTransformer) Finalize() [][]byte {
	var events [][]byte
	if t.started {
		// Ensure we send closing events if not already sent
		return events
	}
	return events
}
