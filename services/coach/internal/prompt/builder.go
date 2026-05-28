package prompt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type MorningInput struct {
	Date        string
	Timezone    string
	WakeAtLocal string
	DataState   string

	MetricsJSON string
}

func BuildMorningPrompt(promptVersion string, path string, input MorningInput) (string, error) {
	promptVersion, err := normalizePromptVersion(promptVersion)
	if err != nil {
		return "", err
	}

	rawTemplate, err := loadTemplate(promptVersion, path)
	if err != nil {
		return "", err
	}

	metricsJSON, err := normalizeMetricsJSON(input.MetricsJSON)
	if err != nil {
		return "", fmt.Errorf("prompt: invalid metrics_json: %w", err)
	}

	tpl, err := template.New(promptVersion).Parse(rawTemplate)
	if err != nil {
		return "", fmt.Errorf("prompt: parse template %q: %w", promptVersion, err)
	}

	data := struct {
		Date        string
		Timezone    string
		WakeAtLocal string
		DataState   string
		MetricsJSON string
	}{
		Date:        strings.TrimSpace(input.Date),
		Timezone:    strings.TrimSpace(input.Timezone),
		WakeAtLocal: strings.TrimSpace(input.WakeAtLocal),
		DataState:   strings.TrimSpace(input.DataState),
		MetricsJSON: metricsJSON,
	}

	if data.Date == "" {
		data.Date = "unknown"
	}
	if data.Timezone == "" {
		data.Timezone = "UTC"
	}
	if data.WakeAtLocal == "" {
		data.WakeAtLocal = "unknown"
	}
	if data.DataState == "" {
		data.DataState = "unknown"
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("prompt: execute template %q: %w", promptVersion, err)
	}

	return strings.TrimSpace(buf.String()), nil
}

func loadTemplate(promptVersion string, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("prompt: read external template: %w", err)
		}

		return string(body), nil
	}

	templateFile := promptVersion + ".tmpl"

	body, err := FS.ReadFile(templateFile)
	if err != nil {
		return "", fmt.Errorf("prompt: read embedded template %q: %w", templateFile, err)
	}

	return string(body), nil
}

func normalizePromptVersion(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("prompt: prompt_version is required")
	}

	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.TrimSuffix(value, ".tmpl")
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("prompt: prompt_version is invalid")
	}

	return value, nil
}

func normalizeMetricsJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}

	if !json.Valid([]byte(raw)) {
		return "", fmt.Errorf("not valid JSON")
	}

	var dst bytes.Buffer
	if err := json.Indent(&dst, []byte(raw), "", "  "); err != nil {
		return "", err
	}

	return dst.String(), nil
}
