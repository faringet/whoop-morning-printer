package art

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"time"
)

const (
	ModeDeterministic = "deterministic"
	ModeRandom        = "random"
)

type Template struct {
	Name string
	Body string
}

type Selection struct {
	Name string
	Text string
}

type PickInput struct {
	Enabled bool

	Mode string

	UserID int64
	Date   time.Time

	Width    int
	MaxLines int

	Salt string
}

type Selector struct {
	templates []Template
}

func NewSelector(templates []Template) *Selector {
	copied := make([]Template, 0, len(templates))

	for _, template := range templates {
		template.Name = strings.TrimSpace(template.Name)
		template.Body = strings.TrimSpace(template.Body)

		if template.Name == "" || template.Body == "" {
			continue
		}

		copied = append(copied, template)
	}

	return &Selector{
		templates: copied,
	}
}

func (s *Selector) Pick(input PickInput) Selection {
	if s == nil || !input.Enabled || len(s.templates) == 0 {
		return Selection{}
	}

	if input.Width <= 0 {
		input.Width = 42
	}
	if input.MaxLines <= 0 {
		input.MaxLines = 8
	}

	candidates := s.candidates(input.MaxLines)
	if len(candidates) == 0 {
		return Selection{}
	}

	index := pickIndex(input, len(candidates))
	template := candidates[index]

	return Selection{
		Name: template.Name,
		Text: formatBody(template.Body, input.Width, input.MaxLines),
	}
}

func (s *Selector) candidates(maxLines int) []Template {
	if maxLines <= 0 {
		return s.templates
	}

	out := make([]Template, 0, len(s.templates))

	for _, template := range s.templates {
		if countNonEmptyLines(template.Body) <= maxLines {
			out = append(out, template)
		}
	}

	return out
}

func pickIndex(input PickInput, size int) int {
	if size <= 1 {
		return 0
	}

	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = ModeDeterministic
	}

	if mode == ModeRandom {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		return rng.Intn(size)
	}

	seed := deterministicSeed(input)

	return int(seed % uint64(size))
}

func deterministicSeed(input PickInput) uint64 {
	h := fnv.New64a()

	date := input.Date
	if date.IsZero() {
		date = time.Now().UTC()
	}

	_, _ = h.Write([]byte(fmt.Sprintf(
		"user=%d|date=%s|salt=%s",
		input.UserID,
		date.UTC().Format("2006-01-02"),
		strings.TrimSpace(input.Salt),
	)))

	return h.Sum64()
}

func formatBody(body string, width int, maxLines int) string {
	lines := normalizeLines(body)

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	for i := range lines {
		lines[i] = centerLine(truncateRunes(lines[i], width), width)
	}

	return strings.Join(lines, "\n")
}

func normalizeLines(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	body = strings.Trim(body, "\n")

	rawLines := strings.Split(body, "\n")

	start := 0
	for start < len(rawLines) && strings.TrimSpace(rawLines[start]) == "" {
		start++
	}

	end := len(rawLines)
	for end > start && strings.TrimSpace(rawLines[end-1]) == "" {
		end--
	}

	lines := make([]string, 0, end-start)
	for _, line := range rawLines[start:end] {
		lines = append(lines, strings.TrimRight(line, " \t"))
	}

	return lines
}

func countNonEmptyLines(body string) int {
	lines := normalizeLines(body)

	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count
}

func centerLine(line string, width int) string {
	if width <= 0 {
		return line
	}

	lineLen := runeLen(line)
	if lineLen >= width {
		return line
	}

	leftPad := (width - lineLen) / 2

	return strings.Repeat(" ", leftPad) + line
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return value
	}

	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}

	return string(runes[:maxRunes])
}

func runeLen(value string) int {
	return len([]rune(value))
}
