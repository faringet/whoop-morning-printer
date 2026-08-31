package advisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/coach/internal/ollama"
)

type LLMClient interface {
	Generate(ctx context.Context, model string, prompt string, opts ollama.GenerateOptions) (string, error)
}

type Config struct {
	Model string

	PromptVersion string
	PromptPath    string

	MaxRetries   int
	RetryBackoff time.Duration

	MaxAdviceRunes int
	MaxMottoRunes  int
}

type Advisor struct {
	log *slog.Logger
	cfg Config
	llm LLMClient
	now func() time.Time
}

type Snapshot struct {
	ID     int64
	UserID int64

	Date      time.Time
	DataState string

	SleepScore    *int
	RecoveryScore *int
	DayStrain     *float64

	SleepMinutes       *int
	SleepNeededMinutes *int
	SleepVsNeedPct     *int

	AwakeMinutes       *int
	LightSleepMinutes  *int
	DeepSleepMinutes   *int
	REMSleepMinutes    *int
	RestorativeMinutes *int

	SleepEfficiencyPct  *float64
	SleepConsistencyPct *float64

	RespiratoryRate     *float64
	HRVRMSSDMS          *float64
	RestingHeartRateBPM *int
	SpO2Pct             *float64
	SkinTempCelsius     *float64

	SourceUpdatedAt *time.Time
}

type BuildInput struct {
	Snapshot Snapshot

	Timezone string
	WakeAt   *time.Time
}

type Advice struct {
	UserID     int64
	SnapshotID int64

	Date time.Time

	Model         string
	PromptVersion string

	DayType    string
	MainSignal string
	AdviceText string
	Motto      string

	PayloadJSON json.RawMessage

	GeneratedAt time.Time
}

type advicePayload struct {
	RenderedText string `json:"rendered_text"`
	Motto        string `json:"motto"`

	Model         string `json:"model"`
	PromptVersion string `json:"prompt_version"`
	DataState     string `json:"data_state"`

	Brief      MorningBrief   `json:"brief"`
	Candidates []LLMResponse  `json:"candidates"`
	Metrics    map[string]any `json:"metrics"`
}

func New(log *slog.Logger, cfg Config, llm LLMClient) (*Advisor, error) {
	if log == nil {
		log = slog.Default()
	}
	if llm == nil {
		return nil, errors.New("advisor: llm client is nil")
	}

	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		return nil, errors.New("advisor: model is required")
	}

	cfg.PromptVersion = strings.TrimSpace(cfg.PromptVersion)
	if cfg.PromptVersion == "" {
		return nil, errors.New("advisor: prompt_version is required")
	}

	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 750 * time.Millisecond
	}
	if cfg.MaxAdviceRunes <= 0 {
		cfg.MaxAdviceRunes = 900
	}
	if cfg.MaxMottoRunes <= 0 {
		cfg.MaxMottoRunes = 180
	}

	return &Advisor{
		log: log.With(
			slog.String("layer", "advisor"),
			slog.String("module", "coach.advisor"),
		),
		cfg: cfg,
		llm: llm,
		now: time.Now,
	}, nil
}

func (a *Advisor) Build(ctx context.Context, input BuildInput) (Advice, error) {
	if a == nil {
		return Advice{}, errors.New("advisor: advisor is nil")
	}

	if err := validateSnapshot(input.Snapshot); err != nil {
		return Advice{}, err
	}

	metrics := buildMetricsMap(input.Snapshot)
	brief := buildMorningBrief(input.Snapshot)

	candidates, err := a.generateWriterCandidates(ctx, brief)
	if err != nil {
		return Advice{}, fmt.Errorf("advisor: generate writer candidates: %w", err)
	}

	response, err := a.generateEditorResponse(ctx, brief, candidates)
	if err != nil {
		return Advice{}, fmt.Errorf("advisor: generate editor response: %w", err)
	}

	response = LimitResponseText(response, a.cfg.MaxAdviceRunes, a.cfg.MaxMottoRunes)

	payload, err := json.Marshal(advicePayload{
		RenderedText:  response.RenderedText,
		Motto:         response.Motto,
		Model:         a.cfg.Model,
		PromptVersion: a.cfg.PromptVersion,
		DataState:     input.Snapshot.DataState,
		Brief:         brief,
		Candidates:    candidates,
		Metrics:       metrics,
	})
	if err != nil {
		return Advice{}, fmt.Errorf("advisor: marshal advice payload: %w", err)
	}

	return Advice{
		UserID:        input.Snapshot.UserID,
		SnapshotID:    input.Snapshot.ID,
		Date:          input.Snapshot.Date,
		Model:         a.cfg.Model,
		PromptVersion: a.cfg.PromptVersion,
		DayType:       dayTypeFromLoad(brief.RecommendedLoad),
		MainSignal:    brief.MainSignal,
		AdviceText:    response.RenderedText,
		Motto:         response.Motto,
		PayloadJSON:   payload,
		GeneratedAt:   a.now().UTC(),
	}, nil
}

func validateSnapshot(snapshot Snapshot) error {
	if snapshot.ID <= 0 {
		return errors.New("advisor: snapshot.id must be > 0")
	}
	if snapshot.UserID <= 0 {
		return errors.New("advisor: snapshot.user_id must be > 0")
	}
	if snapshot.Date.IsZero() {
		return errors.New("advisor: snapshot.date is required")
	}

	return nil
}

func buildMetricsMap(snapshot Snapshot) map[string]any {
	return map[string]any{
		"sleep_score":       snapshot.SleepScore,
		"recovery_score":    snapshot.RecoveryScore,
		"day_strain":        snapshot.DayStrain,
		"sleep_vs_need_pct": snapshot.SleepVsNeedPct,
		"awake_minutes":     snapshot.AwakeMinutes,
	}
}
