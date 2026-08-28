package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type WriterInput struct {
	BriefJSON string
}

func BuildWriterPrompt(input WriterInput) (string, error) {
	briefJSON, err := normalizeMetricsJSON(input.BriefJSON)
	if err != nil {
		return "", fmt.Errorf("prompt: invalid brief_json: %w", err)
	}

	rawTemplate, err := loadTemplate("coach_writer_v1", "")
	if err != nil {
		return "", err
	}

	tpl, err := template.New("coach_writer_v1").Parse(rawTemplate)
	if err != nil {
		return "", fmt.Errorf("prompt: parse writer template: %w", err)
	}

	data := WriterInput{
		BriefJSON: briefJSON,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("prompt: execute writer template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}
