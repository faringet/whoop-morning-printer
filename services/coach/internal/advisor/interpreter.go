package advisor

import "fmt"

type ReadinessLevel string

const (
	ReadinessUnknown     ReadinessLevel = "unknown"
	ReadinessExceptional ReadinessLevel = "exceptional"
	ReadinessVeryHigh    ReadinessLevel = "very_high"
	ReadinessGood        ReadinessLevel = "good"
	ReadinessModerate    ReadinessLevel = "moderate"
	ReadinessLow         ReadinessLevel = "low"
)

type LoadLevel string

const (
	LoadUnknown LoadLevel = "unknown"
	LoadHigh    LoadLevel = "high"
	LoadNormal  LoadLevel = "normal"
	LoadReduced LoadLevel = "reduced"
)

type SleepPerformanceLevel string

const (
	SleepPerformanceUnknown    SleepPerformanceLevel = "unknown"
	SleepPerformanceOptimal    SleepPerformanceLevel = "optimal"
	SleepPerformanceSufficient SleepPerformanceLevel = "sufficient"
	SleepPerformancePoor       SleepPerformanceLevel = "poor"
)

type SleepNeedCoverage string

const (
	SleepNeedUnknown           SleepNeedCoverage = "unknown"
	SleepNeedCoveredWithMargin SleepNeedCoverage = "covered_with_margin"
	SleepNeedNearlyCovered     SleepNeedCoverage = "nearly_covered"
	SleepNeedShortfall         SleepNeedCoverage = "shortfall"
	SleepNeedMajorShortfall    SleepNeedCoverage = "major_shortfall"
)

type MorningBrief struct {
	Readiness         ReadinessLevel        `json:"readiness"`
	RecommendedLoad   LoadLevel             `json:"recommended_load"`
	SleepPerformance  SleepPerformanceLevel `json:"sleep_performance"`
	SleepNeedCoverage SleepNeedCoverage     `json:"sleep_need_coverage"`

	MainSignal       string   `json:"main_signal"`
	SecondarySignals []string `json:"secondary_signals,omitempty"`
	Guardrails       []string `json:"guardrails,omitempty"`
}

func buildMorningBrief(snapshot Snapshot) MorningBrief {
	readiness := classifyReadiness(snapshot.RecoveryScore)
	sleepPerformance := classifySleepPerformance(snapshot.SleepScore)
	sleepNeedCoverage := classifySleepNeedCoverage(snapshot.SleepVsNeedPct)

	return MorningBrief{
		Readiness:         readiness,
		RecommendedLoad:   recommendLoad(readiness, sleepPerformance, sleepNeedCoverage),
		SleepPerformance:  sleepPerformance,
		SleepNeedCoverage: sleepNeedCoverage,
		MainSignal:        buildMainSignal(snapshot.RecoveryScore, readiness),
		SecondarySignals:  buildSecondarySignals(snapshot, sleepPerformance, sleepNeedCoverage),
		Guardrails:        buildGuardrails(snapshot),
	}
}

func classifyReadiness(score *int) ReadinessLevel {
	if score == nil {
		return ReadinessUnknown
	}

	switch {
	case *score >= 90:
		return ReadinessExceptional
	case *score >= 80:
		return ReadinessVeryHigh
	case *score >= 67:
		return ReadinessGood
	case *score >= 34:
		return ReadinessModerate
	default:
		return ReadinessLow
	}
}

func classifySleepPerformance(score *int) SleepPerformanceLevel {
	if score == nil {
		return SleepPerformanceUnknown
	}

	switch {
	case *score >= 85:
		return SleepPerformanceOptimal
	case *score >= 70:
		return SleepPerformanceSufficient
	default:
		return SleepPerformancePoor
	}
}

func classifySleepNeedCoverage(value *int) SleepNeedCoverage {
	if value == nil {
		return SleepNeedUnknown
	}

	switch {
	case *value >= 100:
		return SleepNeedCoveredWithMargin
	case *value >= 85:
		return SleepNeedNearlyCovered
	case *value >= 70:
		return SleepNeedShortfall
	default:
		return SleepNeedMajorShortfall
	}
}

func recommendLoad(readiness ReadinessLevel, sleepPerformance SleepPerformanceLevel, sleepNeedCoverage SleepNeedCoverage) LoadLevel {
	switch readiness {
	case ReadinessExceptional:
		return LoadHigh
	case ReadinessVeryHigh, ReadinessGood:
		if sleepPerformance == SleepPerformancePoor || sleepNeedCoverage == SleepNeedMajorShortfall {
			return LoadNormal
		}
		return LoadHigh
	case ReadinessModerate:
		if sleepPerformance == SleepPerformancePoor && sleepNeedCoverage == SleepNeedMajorShortfall {
			return LoadReduced
		}
		return LoadNormal
	case ReadinessLow:
		return LoadReduced
	default:
		return LoadUnknown
	}
}

func buildMainSignal(score *int, readiness ReadinessLevel) string {
	if score == nil {
		return "recovery_score unavailable; readiness cannot be determined reliably"
	}

	return fmt.Sprintf("recovery_score=%d -> %s readiness", *score, readiness)
}

func buildSecondarySignals(snapshot Snapshot, sleepPerformance SleepPerformanceLevel, sleepNeedCoverage SleepNeedCoverage) []string {
	signals := make([]string, 0, 4)

	if snapshot.SleepScore != nil {
		signals = append(signals, fmt.Sprintf("sleep_score=%d -> %s", *snapshot.SleepScore, sleepPerformance))
	}

	if snapshot.SleepVsNeedPct != nil {
		signals = append(signals, fmt.Sprintf("sleep_vs_need_pct=%d -> %s", *snapshot.SleepVsNeedPct, sleepNeedCoverage))
	}

	if snapshot.AwakeMinutes != nil {
		signals = append(signals, fmt.Sprintf("awake_minutes=%d -> secondary sleep context only", *snapshot.AwakeMinutes))
	}

	if snapshot.DayStrain != nil {
		signals = append(signals, fmt.Sprintf("day_strain=%.2f -> accumulated load, not readiness", *snapshot.DayStrain))
	}

	return signals
}

func buildGuardrails(snapshot Snapshot) []string {
	guardrails := make([]string, 0, 4)

	if snapshot.RecoveryScore != nil && *snapshot.RecoveryScore >= 90 {
		guardrails = append(guardrails, "do not recommend reduced load based only on secondary metrics")
	}

	if snapshot.SleepVsNeedPct != nil && *snapshot.SleepVsNeedPct >= 100 {
		guardrails = append(guardrails, "do not describe sleep as oversleeping or excessive")
	}

	if snapshot.AwakeMinutes != nil {
		guardrails = append(guardrails, "do not infer symptoms or diagnoses from awake_minutes")
	}

	if snapshot.DayStrain != nil {
		guardrails = append(guardrails, "do not use day_strain as a readiness score")
	}

	return guardrails
}

func dayTypeFromLoad(load LoadLevel) string {
	switch load {
	case LoadHigh:
		return "push"
	case LoadNormal:
		return "balanced"
	case LoadReduced:
		return "easy"
	default:
		return "unknown"
	}
}
