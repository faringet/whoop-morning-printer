package fieldnote

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxFieldNoteRunes = 160

func normalize(text string) string {
	text = strings.TrimSpace(text)

	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return text
}

func validate(text string) error {
	if text == "" {
		return errors.New("fieldnote: text is empty")
	}

	if !utf8.ValidString(text) {
		return errors.New("fieldnote: text is not valid utf-8")
	}

	runeCount := utf8.RuneCountInString(text)
	if runeCount > maxFieldNoteRunes {
		return errors.New("fieldnote: text is too long")
	}

	for _, r := range text {
		if unicode.IsControl(r) {
			return errors.New("fieldnote: text contains control characters")
		}
	}

	if strings.Contains(text, "```") {
		return errors.New("fieldnote: text contains markdown code block")
	}

	return nil
}
