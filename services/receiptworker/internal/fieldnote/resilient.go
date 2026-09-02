package fieldnote

import (
	"context"
	"errors"
	"fmt"
)

type ResilientGenerator struct {
	generators []Generator
}

func NewResilientGenerator(generators ...Generator) *ResilientGenerator {
	return &ResilientGenerator{
		generators: generators,
	}
}

func (g *ResilientGenerator) Generate(ctx context.Context, input Input) (Result, error) {
	if g == nil {
		return Result{}, errors.New("fieldnote resilient: generator is nil")
	}

	if len(g.generators) == 0 {
		return Result{}, errors.New("fieldnote resilient: generators are empty")
	}

	var lastErr error

	for _, generator := range g.generators {
		if generator == nil {
			continue
		}

		result, err := generator.Generate(ctx, input)
		if err != nil {
			lastErr = err
			continue
		}

		result.Text = normalize(result.Text)

		err = validate(result.Text)
		if err != nil {
			lastErr = fmt.Errorf("fieldnote resilient: invalid result: %w", err)
			continue
		}

		return result, nil
	}

	if lastErr != nil {
		return Result{}, fmt.Errorf("fieldnote resilient: all generators failed: %w", lastErr)
	}

	return Result{}, errors.New("fieldnote resilient: no generators available")
}
