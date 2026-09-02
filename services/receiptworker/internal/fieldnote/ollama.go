package fieldnote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const ollamaFieldNotePrompt = `Придумай одну короткую мысль на день на русском языке.

Требования:
- максимум 120 символов
- одно предложение
- без приветствий
- без кавычек
- без markdown
- без emoji
- без медицинских советов
- без пафосной мотивации
- не повторяй пункты чек-листа
- стиль спокойный, сухой, слегка ироничный
- можно использовать компьютерные или инженерные метафоры
- верни только сам текст`

type OllamaGenerator struct {
	baseURL   string
	model     string
	keepAlive string
	client    *http.Client
}

type ollamaGenerateRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	Stream    bool   `json:"stream"`
	Think     bool   `json:"think"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaGenerator(baseURL string, model string, keepAlive string, timeout time.Duration) *OllamaGenerator {
	client := &http.Client{
		Timeout: timeout,
	}

	return &OllamaGenerator{
		baseURL:   strings.TrimRight(baseURL, "/"),
		model:     strings.TrimSpace(model),
		keepAlive: strings.TrimSpace(keepAlive),
		client:    client,
	}
}

func (g *OllamaGenerator) Generate(ctx context.Context, _ Input) (Result, error) {
	if g == nil {
		return Result{}, errors.New("fieldnote ollama: generator is nil")
	}

	if g.client == nil {
		return Result{}, errors.New("fieldnote ollama: client is nil")
	}

	if g.baseURL == "" {
		return Result{}, errors.New("fieldnote ollama: base_url is empty")
	}

	if g.model == "" {
		return Result{}, errors.New("fieldnote ollama: model is empty")
	}

	payload := ollamaGenerateRequest{
		Model:     g.model,
		Prompt:    ollamaFieldNotePrompt,
		Stream:    false,
		Think:     false,
		KeepAlive: g.keepAlive,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, fmt.Errorf("fieldnote ollama: marshal request: %w", err)
	}

	url := g.baseURL + "/api/generate"

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("fieldnote ollama: create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := g.client.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("fieldnote ollama: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return Result{}, fmt.Errorf("fieldnote ollama: unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var ollamaResponse ollamaGenerateResponse

	err = json.NewDecoder(response.Body).Decode(&ollamaResponse)
	if err != nil {
		return Result{}, fmt.Errorf("fieldnote ollama: decode response: %w", err)
	}

	text := strings.TrimSpace(ollamaResponse.Response)
	if text == "" {
		return Result{}, errors.New("fieldnote ollama: response is empty")
	}

	result := Result{
		Text:   text,
		Source: SourceOllama,
	}

	return result, nil
}
