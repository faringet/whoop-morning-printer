package advisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type DayType string

const (
	DayTypeRecovery DayType = "recovery"
	DayTypeEasy     DayType = "easy"
	DayTypeBalanced DayType = "balanced"
	DayTypePush     DayType = "push"
	DayTypeRest     DayType = "rest"
	DayTypeUnknown  DayType = "unknown"
)

type TrainingIntensity string

const (
	TrainingIntensityNone     TrainingIntensity = "none"
	TrainingIntensityLow      TrainingIntensity = "low"
	TrainingIntensityModerate TrainingIntensity = "moderate"
	TrainingIntensityHigh     TrainingIntensity = "high"
)

type RiskLevel string

const (
	RiskLevelLow     RiskLevel = "low"
	RiskLevelMedium  RiskLevel = "medium"
	RiskLevelHigh    RiskLevel = "high"
	RiskLevelUnknown RiskLevel = "unknown"
)

type LLMResponse struct {
	DayType           DayType           `json:"day_type"`
	MainSignal        string            `json:"main_signal"`
	AdviceText        string            `json:"advice_text"`
	Motto             string            `json:"motto"`
	TrainingIntensity TrainingIntensity `json:"training_intensity"`
	RiskLevel         RiskLevel         `json:"risk_level"`
	FocusHint         string            `json:"focus_hint"`
	RecoveryHint      string            `json:"recovery_hint"`
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

	response.DayType = normalizeDayType(response.DayType)
	response.TrainingIntensity = normalizeTrainingIntensity(response.TrainingIntensity)
	response.RiskLevel = normalizeRiskLevel(response.RiskLevel)

	response.MainSignal = normalizeOneLine(response.MainSignal)
	response.AdviceText = normalizeText(response.AdviceText)
	response.Motto = normalizeOneLine(response.Motto)
	response.FocusHint = normalizeOneLine(response.FocusHint)
	response.RecoveryHint = normalizeOneLine(response.RecoveryHint)

	if response.DayType == "" {
		return LLMResponse{}, errors.New("advisor: day_type is invalid")
	}
	if response.TrainingIntensity == "" {
		return LLMResponse{}, errors.New("advisor: training_intensity is invalid")
	}
	if response.RiskLevel == "" {
		return LLMResponse{}, errors.New("advisor: risk_level is invalid")
	}
	if response.MainSignal == "" {
		return LLMResponse{}, errors.New("advisor: main_signal is empty")
	}
	if response.AdviceText == "" {
		return LLMResponse{}, errors.New("advisor: advice_text is empty")
	}
	if response.Motto == "" {
		return LLMResponse{}, errors.New("advisor: motto is empty")
	}

	return response, nil
}

func normalizeDayType(value DayType) DayType {
	switch strings.TrimSpace(strings.ToLower(string(value))) {
	case "recovery":
		return DayTypeRecovery
	case "easy":
		return DayTypeEasy
	case "balanced", "balance", "normal":
		return DayTypeBalanced
	case "push":
		return DayTypePush
	case "rest":
		return DayTypeRest
	case "unknown":
		return DayTypeUnknown
	default:
		return ""
	}
}

func normalizeTrainingIntensity(value TrainingIntensity) TrainingIntensity {
	switch strings.TrimSpace(strings.ToLower(string(value))) {
	case "none", "no", "zero":
		return TrainingIntensityNone
	case "low", "easy":
		return TrainingIntensityLow
	case "moderate", "medium":
		return TrainingIntensityModerate
	case "high", "hard":
		return TrainingIntensityHigh
	default:
		return ""
	}
}

func normalizeRiskLevel(value RiskLevel) RiskLevel {
	switch strings.TrimSpace(strings.ToLower(string(value))) {
	case "low":
		return RiskLevelLow
	case "medium", "moderate":
		return RiskLevelMedium
	case "high":
		return RiskLevelHigh
	case "unknown":
		return RiskLevelUnknown
	default:
		return ""
	}
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
