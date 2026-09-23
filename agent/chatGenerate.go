package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const DefaultModel = "liquid/lfm-2.5-2.6b:free"

var GeneratorModel = DefaultModel

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResult struct {
	RawOutput        string `json:"raw_output"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	ReasoningDetails any    `json:"reasoning_details,omitempty"`
}

var (
	GlobalAIKey   string
	GlobalBaseURL string
)

// SetAIConfig sets global AI API credentials programmatically.
func SetAIConfig(apiKey, baseURL string) {
	if apiKey != "" {
		GlobalAIKey = apiKey
	}
	if baseURL != "" {
		GlobalBaseURL = baseURL
	}
}

// ChatGenerate calls the OpenRouter/OpenAI chat completion API with retry & backoff.
func ChatGenerate(ctx context.Context, messages []Message, tools any, maxNewTokens int, model string, retries int) (*ChatResult, error) {
	_ = godotenv.Load(".env")

	if model == "" {
		model = GeneratorModel
	}
	if maxNewTokens <= 0 {
		maxNewTokens = 4096
	}

	baseURL := GlobalBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("BASE_AI_URL")
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	apiKey := GlobalAIKey
	if apiKey == "" {
		apiKey = os.Getenv("AI_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	reqBody := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxNewTokens,
		"extra_body": map[string]any{"reasoning": map[string]any{"enabled": true}},
	}
	if tools != nil {
		reqBody["tools"] = tools
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 90 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()

			var res struct {
				Choices []struct {
					Message struct {
						Content          string `json:"content"`
						ReasoningContent string `json:"reasoning_content"`
						ReasoningDetails any    `json:"reasoning_details"`
						Reasoning        any    `json:"reasoning"`
					} `json:"message"`
				} `json:"choices"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}

			if decodeErr := json.NewDecoder(resp.Body).Decode(&res); decodeErr == nil {
				if resp.StatusCode == http.StatusOK {
					var rawOutput string
					var reasoning any

					if len(res.Choices) > 0 {
						rawOutput = res.Choices[0].Message.Content
						if strings.TrimSpace(rawOutput) == "" && strings.TrimSpace(res.Choices[0].Message.ReasoningContent) != "" {
							rawOutput = res.Choices[0].Message.ReasoningContent
						}
						reasoning = res.Choices[0].Message.ReasoningDetails
						if reasoning == nil {
							reasoning = res.Choices[0].Message.Reasoning
						}
						if reasoning == nil {
							reasoning = res.Choices[0].Message.ReasoningContent
						}
					}

					return &ChatResult{
						RawOutput:        rawOutput,
						InputTokens:      res.Usage.PromptTokens,
						OutputTokens:     res.Usage.CompletionTokens,
						ReasoningDetails: reasoning,
					}, nil
				}
				err = fmt.Errorf("api error (%d): %s", resp.StatusCode, res.Error.Message)
			} else {
				err = decodeErr
			}
		}

		lastErr = err
		wait := computeRetryWait(resp, attempt)

		if attempt < retries {
			log.Printf("[chat_generate] attempt %d/%d failed (%v), retrying in %v", attempt+1, retries+1, err, wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		} else {
			log.Printf("[chat_generate] all %d attempts failed: %v", retries+1, err)
		}
	}

	return nil, lastErr
}

func computeRetryWait(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if sec, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return time.Duration(1<<attempt) * time.Second
}
