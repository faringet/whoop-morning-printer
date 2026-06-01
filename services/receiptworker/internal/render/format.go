package render

import (
	"fmt"
	"strings"
	"time"
)

const defaultWidth = 42

type Builder struct {
	width         int
	lineSeparator string
	lines         []string
}

func NewBuilder(width int, lineSeparator string) *Builder {
	if width <= 0 {
		width = defaultWidth
	}

	lineSeparator = strings.TrimSpace(lineSeparator)
	if lineSeparator == "" {
		lineSeparator = "-"
	}

	return &Builder{
		width:         width,
		lineSeparator: lineSeparator,
		lines:         make([]string, 0, 64),
	}
}

func (b *Builder) Width() int {
	if b == nil || b.width <= 0 {
		return defaultWidth
	}

	return b.width
}

func (b *Builder) Line(value string) {
	if b == nil {
		return
	}

	value = strings.TrimRight(value, " \t\r\n")
	b.lines = append(b.lines, truncateRunes(value, b.Width()))
}

func (b *Builder) Raw(value string) {
	if b == nil {
		return
	}

	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	for _, line := range strings.Split(value, "\n") {
		b.Line(line)
	}
}

func (b *Builder) Text(value string) {
	if b == nil {
		return
	}

	value = normalizeSpaces(value)
	if value == "" {
		return
	}

	for _, line := range wrapText(value, b.Width()) {
		b.Line(line)
	}
}

func (b *Builder) Blank() {
	if b == nil {
		return
	}

	b.lines = append(b.lines, "")
}

func (b *Builder) Separator() {
	if b == nil {
		return
	}

	sep := b.lineSeparator
	if sep == "" {
		sep = "-"
	}

	b.Line(repeatToWidth(sep, b.Width()))
}

func (b *Builder) Title(value string) {
	if b == nil {
		return
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	b.Separator()
	b.Center(value)
	b.Separator()
}

func (b *Builder) Center(value string) {
	if b == nil {
		return
	}

	value = strings.TrimSpace(value)
	if value == "" {
		b.Blank()
		return
	}

	for _, line := range wrapText(value, b.Width()) {
		b.Line(centerLine(line, b.Width()))
	}
}

func (b *Builder) KeyValue(key string, value string) {
	if b == nil {
		return
	}

	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)

	if key == "" && value == "" {
		return
	}

	if key == "" {
		b.Text(value)
		return
	}

	if value == "" {
		value = "-"
	}

	prefix := key + ": "
	available := b.Width() - runeLen(prefix)

	if available <= 0 {
		b.Line(truncateRunes(key, b.Width()))
		b.Text(value)
		return
	}

	wrappedValue := wrapText(value, available)
	if len(wrappedValue) == 0 {
		b.Line(prefix + "-")
		return
	}

	b.Line(prefix + wrappedValue[0])

	indent := strings.Repeat(" ", runeLen(prefix))
	for _, line := range wrappedValue[1:] {
		b.Line(indent + line)
	}
}

func (b *Builder) String() string {
	if b == nil {
		return ""
	}

	lines := make([]string, 0, len(b.lines))

	for _, line := range b.lines {
		lines = append(lines, strings.TrimRight(line, " \t"))
	}

	out := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if out == "" {
		return ""
	}

	return out + "\n"
}

func FormatLocalDate(t time.Time, timezone string) string {
	if t.IsZero() {
		return "-"
	}

	return t.In(loadLocationOrUTC(timezone)).Format("02.01.2006")
}

func FormatLocalTime(t time.Time, timezone string) string {
	if t.IsZero() {
		return "-"
	}

	return t.In(loadLocationOrUTC(timezone)).Format("15:04")
}

func FormatLocalDateTime(t time.Time, timezone string) string {
	if t.IsZero() {
		return "-"
	}

	return t.In(loadLocationOrUTC(timezone)).Format("02.01.2006 15:04")
}

func FormatIntPtr(value *int, suffix string) string {
	if value == nil {
		return "-"
	}

	return strings.TrimSpace(fmt.Sprintf("%d %s", *value, suffix))
}

func FormatFloatPtr(value *float64, precision int, suffix string) string {
	if value == nil {
		return "-"
	}

	if precision < 0 {
		precision = 1
	}

	format := fmt.Sprintf("%%.%df %%s", precision)

	return strings.TrimSpace(fmt.Sprintf(format, *value, suffix))
}

func FormatMinutesPtr(value *int) string {
	if value == nil {
		return "-"
	}

	minutes := *value
	if minutes < 0 {
		minutes = 0
	}

	hours := minutes / 60
	leftMinutes := minutes % 60

	if hours <= 0 {
		return fmt.Sprintf("%d min", leftMinutes)
	}

	return fmt.Sprintf("%dh %02dm", hours, leftMinutes)
}

func normalizeSpaces(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.TrimSpace(value)

	return strings.Join(strings.Fields(value), " ")
}

func wrapText(value string, width int) []string {
	value = normalizeSpaces(value)
	if value == "" {
		return nil
	}

	if width <= 0 {
		width = defaultWidth
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return nil
	}

	lines := make([]string, 0, 4)
	current := ""

	for _, word := range words {
		if runeLen(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}

			for runeLen(word) > width {
				lines = append(lines, truncateRunes(word, width))
				word = string([]rune(word)[width:])
			}

			if word != "" {
				current = word
			}

			continue
		}

		if current == "" {
			current = word
			continue
		}

		next := current + " " + word
		if runeLen(next) <= width {
			current = next
			continue
		}

		lines = append(lines, current)
		current = word
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func centerLine(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 {
		return value
	}

	value = truncateRunes(value, width)

	valueLen := runeLen(value)
	if valueLen >= width {
		return value
	}

	leftPad := (width - valueLen) / 2

	return strings.Repeat(" ", leftPad) + value
}

func repeatToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}

	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}

	var b strings.Builder
	for runeLen(b.String()) < width {
		b.WriteString(value)
	}

	return truncateRunes(b.String(), width)
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

func loadLocationOrUTC(timezone string) *time.Location {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return time.UTC
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}

	return loc
}
