package fieldnote

import (
	"context"
	"time"
)

type Source string

const (
	SourceOllama    Source = "ollama"
	SourceGrammar   Source = "grammar"
	SourceEmergency Source = "emergency"
)

type Input struct {
	UserID     int64
	WakePlanID int64
	Date       time.Time
}

type Result struct {
	Text   string
	Source Source
}

type Generator interface {
	Generate(ctx context.Context, input Input) (Result, error)
}
