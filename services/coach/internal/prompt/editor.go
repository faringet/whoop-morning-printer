package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type EditorInput struct {
	BriefJSON      string
	CandidatesJSON string
}

func BuildEditorPrompt(input EditorInput) (string, error) {
	briefJSON, err := normalizeMetricsJSON(input.BriefJSON)
	if err != nil {
		return "", fmt.Errorf("prompt: invalid brief_json: %w", err)
	}

	candidatesJSON, err := normalizeMetricsJSON(input.CandidatesJSON)
	if err != nil {
		return "", fmt.Errorf("prompt: invalid candidates_json: %w", err)
	}

	rawTemplate, err := loadTemplate("coach_editor_v1", "")
	if err != nil {
		return "", err
	}

	tpl, err := template.New("coach_editor_v1").Parse(rawTemplate)
	if err != nil {
		return "", fmt.Errorf("prompt: parse editor template: %w", err)
	}

	data := EditorInput{
		BriefJSON:      briefJSON,
		CandidatesJSON: candidatesJSON,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("prompt: execute editor template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}
