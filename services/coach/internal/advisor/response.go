package advisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type LLMResponse struct {
	RenderedText string `json:"rendered_text"`
	Motto        string `json:"motto"`
}

func ParseLLMResponse(raw string) (LLMResponse, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return LLMResponse{}, errors.New("advisor: empty LLM response")
	}

	jsonObject := extractFirstJSONObject(raw)
	if jsonObject == "" {
		return LLMResponse{}, errors.New("advisor: no JSON object found in LLM response")
	}

	var response LLMResponse
	if err := json.Unmarshal([]byte(jsonObject), &response); err != nil {
		return LLMResponse{}, fmt.Errorf("advisor: unmarshal LLM JSON: %w", err)
	}

	response.RenderedText = normalizeText(response.RenderedText)
	response.Motto = normalizeOneLine(response.Motto)

	if response.RenderedText == "" {
		return LLMResponse{}, errors.New("advisor: rendered_text is empty")
	}
	if response.Motto == "" {
		return LLMResponse{}, errors.New("advisor: motto is empty")
	}

	return response, nil
}

func extractFirstJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(raw); i++ {
		ch := raw[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++

		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(raw[start : i+1])
			}
		}
	}

	return ""
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.TrimSpace(value)

	return strings.Join(strings.Fields(value), " ")
}

func normalizeOneLine(value string) string {
	value = normalizeText(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")

	return strings.Join(strings.Fields(value), " ")
}
