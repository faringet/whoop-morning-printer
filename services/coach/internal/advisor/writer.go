package advisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/faringet/whoop-morning-printer/services/coach/internal/ollama"
	coachprompt "github.com/faringet/whoop-morning-printer/services/coach/internal/prompt"
)

const writerCandidateCount = 4

type writerResponse struct {
	Candidates []LLMResponse `json:"candidates"`
}

func (a *Advisor) generateWriterCandidates(ctx context.Context, brief MorningBrief) ([]LLMResponse, error) {
	briefJSON, err := json.Marshal(brief)
	if err != nil {
		return nil, fmt.Errorf("advisor writer: marshal brief: %w", err)
	}

	promptText, err := coachprompt.BuildWriterPrompt(coachprompt.WriterInput{BriefJSON: string(briefJSON)})
	if err != nil {
		return nil, fmt.Errorf("advisor writer: build prompt: %w", err)
	}

	opts := ollama.GenerateOptions{Temperature: 0.7, TopP: 0.92}

	lastErr := errors.New("advisor writer: generation failed")

	for attempt := 0; attempt <= a.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleep := a.cfg.RetryBackoff * time.Duration(attempt)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleep):
			}
		}

		raw, err := a.llm.Generate(ctx, a.cfg.Model, promptText, opts)
		if err != nil {
			lastErr = fmt.Errorf("ollama generate: %w", err)

			a.log.Warn("writer generate failed",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", a.cfg.MaxRetries+1),
				slog.Any("err", err),
			)

			continue
		}

		candidates, err := parseWriterResponse(raw)
		if err != nil {
			lastErr = err

			a.log.Warn("writer response parse failed",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", a.cfg.MaxRetries+1),
				slog.String("raw_snippet", safeSnippet(raw, 300)),
				slog.Any("err", err),
			)

			continue
		}

		return candidates, nil
	}

	return nil, lastErr
}

func parseWriterResponse(raw string) ([]LLMResponse, error) {
	jsonObject := extractFirstJSONObject(raw)
	if jsonObject == "" {
		return nil, errors.New("advisor writer: no JSON object found")
	}

	var response writerResponse
	if err := json.Unmarshal([]byte(jsonObject), &response); err != nil {
		return nil, fmt.Errorf("advisor writer: unmarshal response: %w", err)
	}

	if len(response.Candidates) != writerCandidateCount {
		return nil, fmt.Errorf("advisor writer: got %d candidates, want %d", len(response.Candidates), writerCandidateCount)
	}

	for i := range response.Candidates {
		response.Candidates[i].RenderedText = normalizeText(response.Candidates[i].RenderedText)
		response.Candidates[i].Motto = normalizeOneLine(response.Candidates[i].Motto)

		if response.Candidates[i].RenderedText == "" {
			return nil, fmt.Errorf("advisor writer: candidate %d rendered_text is empty", i+1)
		}

		if response.Candidates[i].Motto == "" {
			return nil, fmt.Errorf("advisor writer: candidate %d motto is empty", i+1)
		}
	}

	return response.Candidates, nil
}
