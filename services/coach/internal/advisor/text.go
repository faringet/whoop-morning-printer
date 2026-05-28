package advisor

import "strings"

const ellipsis = "…"

func LimitResponseText(response LLMResponse, maxAdviceRunes int, maxMottoRunes int) LLMResponse {
	if maxAdviceRunes <= 0 {
		maxAdviceRunes = 900
	}
	if maxMottoRunes <= 0 {
		maxMottoRunes = 180
	}

	response.MainSignal = truncateRunes(normalizeOneLine(response.MainSignal), 120)
	response.AdviceText = truncateRunes(normalizeText(response.AdviceText), maxAdviceRunes)
	response.Motto = truncateRunes(normalizeOneLine(response.Motto), maxMottoRunes)
	response.FocusHint = truncateRunes(normalizeOneLine(response.FocusHint), 180)
	response.RecoveryHint = truncateRunes(normalizeOneLine(response.RecoveryHint), 180)

	return response
}

func safeSnippet(value string, maxRunes int) string {
	value = normalizeText(value)
	if value == "" {
		return ""
	}

	return truncateRunes(value, maxRunes)
}

func truncateRunes(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return value
	}

	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}

	if maxRunes <= len([]rune(ellipsis)) {
		return string(runes[:maxRunes])
	}

	return strings.TrimSpace(string(runes[:maxRunes-len([]rune(ellipsis))])) + ellipsis
}
