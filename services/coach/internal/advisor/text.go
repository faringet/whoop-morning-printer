package advisor

import "strings"

const ellipsis = "…"

func LimitResponseText(response LLMResponse, maxRenderedTextRunes int, maxMottoRunes int) LLMResponse {
	if maxRenderedTextRunes <= 0 {
		maxRenderedTextRunes = 900
	}
	if maxMottoRunes <= 0 {
		maxMottoRunes = 180
	}

	response.RenderedText = truncateRunes(normalizeText(response.RenderedText), maxRenderedTextRunes)
	response.Motto = truncateRunes(normalizeOneLine(response.Motto), maxMottoRunes)

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
