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

func (a *Advisor) generateEditorResponse(ctx context.Context, brief MorningBrief, candidates []LLMResponse) (LLMResponse, error) {
	if len(candidates) != writerCandidateCount {
		return LLMResponse{}, fmt.Errorf("advisor editor: got %d candidates, want %d", len(candidates), writerCandidateCount)
	}

	briefJSON, err := json.Marshal(brief)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("advisor editor: marshal brief: %w", err)
	}

	candidatesJSON, err := json.Marshal(candidates)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("advisor editor: marshal candidates: %w", err)
	}

	promptText, err := coachprompt.BuildEditorPrompt(coachprompt.EditorInput{BriefJSON: string(briefJSON), CandidatesJSON: string(candidatesJSON)})
	if err != nil {
		return LLMResponse{}, fmt.Errorf("advisor editor: build prompt: %w", err)
	}

	opts := ollama.GenerateOptions{Temperature: 0.15, TopP: 0.90}

	var lastErr error

	for attempt := 0; attempt <= a.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			sleep := a.cfg.RetryBackoff * time.Duration(attempt)

			select {
			case <-ctx.Done():
				return LLMResponse{}, ctx.Err()
			case <-time.After(sleep):
			}
		}

		raw, err := a.llm.Generate(ctx, a.cfg.Model, promptText, opts)
		if err != nil {
			lastErr = fmt.Errorf("ollama generate: %w", err)

			a.log.Warn("editor generate failed",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", a.cfg.MaxRetries+1),
				slog.Any("err", err),
			)

			continue
		}

		response, err := ParseLLMResponse(raw)
		if err != nil {
			lastErr = fmt.Errorf("parse editor response: %w", err)

			a.log.Warn("editor response parse failed",
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", a.cfg.MaxRetries+1),
				slog.String("raw_snippet", safeSnippet(raw, 300)),
				slog.Any("err", err),
			)

			continue
		}

		return response, nil
	}

	if lastErr == nil {
		lastErr = errors.New("advisor editor: generation failed")
	}

	return LLMResponse{}, lastErr
}
