package fieldnote

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"unicode"
)

type GrammarGenerator struct{}

func NewGrammarGenerator() *GrammarGenerator {
	return &GrammarGenerator{}
}

func (g *GrammarGenerator) Generate(ctx context.Context, input Input) (Result, error) {
	err := validateInput(input)
	if err != nil {
		return Result{}, err
	}

	err = ctx.Err()
	if err != nil {
		return Result{}, fmt.Errorf("fieldnote grammar: %w", err)
	}

	seed := grammarSeed(input)
	source := rand.NewSource(seed)
	rng := rand.New(source)

	generators := []func(*rand.Rand) string{
		generateActionNote,
		generateAttentionNote,
		generateTradeoffNote,
		generateSystemNote,
		generateReminderNote,
	}

	generatorIndex := rng.Intn(len(generators))
	generator := generators[generatorIndex]

	text := generator(rng)
	text = strings.TrimSpace(text)

	if text == "" {
		return Result{}, errors.New("fieldnote grammar: generated text is empty")
	}

	result := Result{
		Text:   text,
		Source: SourceGrammar,
	}

	return result, nil
}

func generateActionNote(rng *rand.Rand) string {
	action := pick(rng, actionPhrases)
	timing := pick(rng, actionTimings)

	text := fmt.Sprintf("%s %s.", action, timing)

	return text
}

func generateAttentionNote(rng *rand.Rand) string {
	opening := pick(rng, attentionOpenings)
	action := pick(rng, attentionActions)

	text := fmt.Sprintf("%s, %s.", opening, action)

	return text
}

func generateTradeoffNote(rng *rand.Rand) string {
	subject := pick(rng, tradeoffSubjects)
	comparison := pick(rng, tradeoffComparisons)

	text := fmt.Sprintf("%s %s.", subject, comparison)

	return text
}

func generateSystemNote(rng *rand.Rand) string {
	observation := pick(rng, systemObservations)
	consequence := pick(rng, systemConsequences)

	consequence = upperFirst(consequence)

	text := fmt.Sprintf("%s. %s.", observation, consequence)

	return text
}

func generateReminderNote(rng *rand.Rand) string {
	note := pick(rng, reminderNotes)

	return note
}

func pick(rng *rand.Rand, values []string) string {
	if len(values) == 0 {
		return ""
	}

	index := rng.Intn(len(values))
	value := values[index]

	return value
}

func validateInput(input Input) error {
	if input.UserID <= 0 {
		return errors.New("fieldnote grammar: user_id must be > 0")
	}

	if input.WakePlanID <= 0 {
		return errors.New("fieldnote grammar: wake_plan_id must be > 0")
	}

	if input.Date.IsZero() {
		return errors.New("fieldnote grammar: date is required")
	}

	return nil
}

func grammarSeed(input Input) int64 {
	hash := fnv.New64a()

	date := input.Date.UTC().Format("2006-01-02")
	value := fmt.Sprintf("user=%d|wake_plan=%d|date=%s", input.UserID, input.WakePlanID, date)

	data := []byte(value)
	_, _ = hash.Write(data)

	seed := int64(hash.Sum64())

	return seed
}

func upperFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}

	runes[0] = unicode.ToUpper(runes[0])

	result := string(runes)

	return result
}
