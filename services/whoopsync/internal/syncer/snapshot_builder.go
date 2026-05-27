package syncer

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/storage"
	"github.com/faringet/whoop-morning-printer/services/whoopsync/internal/whoopapi"
)

const defaultTimezone = "Europe/Moscow"

type SnapshotBuildInput struct {
	UserID   int64
	Date     time.Time
	Now      time.Time
	Timezone string

	Sleeps     []whoopapi.SleepRecord
	Recoveries []whoopapi.RecoveryRecord
	Cycles     []whoopapi.CycleRecord
	Workouts   []whoopapi.WorkoutRecord
}

type SnapshotBuildResult struct {
	Snapshot   storage.DailyHealthSnapshot
	RawObjects []storage.RawWHOOPObject
}

func BuildSnapshot(input SnapshotBuildInput) (SnapshotBuildResult, error) {
	if input.UserID <= 0 {
		return SnapshotBuildResult{}, fmt.Errorf("snapshot builder: user_id must be > 0")
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()

	loc := loadLocation(input.Timezone)

	sleep := selectMainSleep(input.Sleeps)
	recovery := selectRecovery(input.Recoveries, sleep)
	cycle := selectCycle(input.Cycles, sleep, recovery)

	date := normalizeSnapshotDate(input.Date, now, loc, sleep)

	snapshot := storage.DailyHealthSnapshot{
		UserID:    input.UserID,
		Date:      date,
		DataState: storage.DataStatePending,
	}

	applySleep(&snapshot, sleep)
	applyRecovery(&snapshot, recovery)
	applyCycle(&snapshot, cycle)
	snapshot.SourceUpdatedAt = latestUpdatedAt(sleepUpdatedAt(sleep), recoveryUpdatedAt(recovery), cycleUpdatedAt(cycle))

	snapshot.DataState = detectDataState(snapshot, sleep, recovery, cycle)

	rawObjects := make([]storage.RawWHOOPObject, 0, len(input.Sleeps)+len(input.Recoveries)+len(input.Cycles)+len(input.Workouts))
	rawObjects = append(rawObjects, buildSleepRawObjects(input.UserID, now, input.Sleeps)...)
	rawObjects = append(rawObjects, buildRecoveryRawObjects(input.UserID, now, input.Recoveries)...)
	rawObjects = append(rawObjects, buildCycleRawObjects(input.UserID, now, input.Cycles)...)
	rawObjects = append(rawObjects, buildWorkoutRawObjects(input.UserID, now, input.Workouts)...)

	return SnapshotBuildResult{
		Snapshot:   snapshot,
		RawObjects: rawObjects,
	}, nil
}

func selectMainSleep(sleeps []whoopapi.SleepRecord) *whoopapi.SleepRecord {
	var selected *whoopapi.SleepRecord

	for i := range sleeps {
		sleep := &sleeps[i]
		if sleep.Sleep.Nap {
			continue
		}

		if selected == nil || sleep.Sleep.End.After(selected.Sleep.End) {
			selected = sleep
		}
	}

	if selected != nil {
		return selected
	}

	for i := range sleeps {
		sleep := &sleeps[i]
		if selected == nil || sleep.Sleep.End.After(selected.Sleep.End) {
			selected = sleep
		}
	}

	return selected
}

func selectRecovery(recoveries []whoopapi.RecoveryRecord, sleep *whoopapi.SleepRecord) *whoopapi.RecoveryRecord {
	if len(recoveries) == 0 {
		return nil
	}

	if sleep != nil {
		for i := range recoveries {
			recovery := &recoveries[i]
			if recovery.Recovery.SleepID != "" && recovery.Recovery.SleepID == sleep.Sleep.ID {
				return recovery
			}
		}

		for i := range recoveries {
			recovery := &recoveries[i]
			if recovery.Recovery.CycleID != 0 && recovery.Recovery.CycleID == sleep.Sleep.CycleID {
				return recovery
			}
		}
	}

	var selected *whoopapi.RecoveryRecord
	for i := range recoveries {
		recovery := &recoveries[i]
		if selected == nil || recovery.Recovery.UpdatedAt.After(selected.Recovery.UpdatedAt) {
			selected = recovery
		}
	}

	return selected
}

func selectCycle(cycles []whoopapi.CycleRecord, sleep *whoopapi.SleepRecord, recovery *whoopapi.RecoveryRecord) *whoopapi.CycleRecord {
	if len(cycles) == 0 {
		return nil
	}

	if sleep != nil && sleep.Sleep.CycleID != 0 {
		for i := range cycles {
			cycle := &cycles[i]
			if cycle.Cycle.ID == sleep.Sleep.CycleID {
				return cycle
			}
		}
	}

	if recovery != nil && recovery.Recovery.CycleID != 0 {
		for i := range cycles {
			cycle := &cycles[i]
			if cycle.Cycle.ID == recovery.Recovery.CycleID {
				return cycle
			}
		}
	}

	var selected *whoopapi.CycleRecord
	for i := range cycles {
		cycle := &cycles[i]
		if selected == nil || cycle.Cycle.End.After(selected.Cycle.End) {
			selected = cycle
		}
	}

	return selected
}

func applySleep(snapshot *storage.DailyHealthSnapshot, sleep *whoopapi.SleepRecord) {
	if snapshot == nil || sleep == nil {
		return
	}

	snapshot.SleepWHOOPID = stringPtr(sleep.Sleep.ID)

	if sleep.Sleep.ScoreState != whoopapi.ScoreStateScored || sleep.Sleep.Score == nil {
		return
	}

	score := sleep.Sleep.Score
	stages := score.StageSummary

	sleepMinutes := millisToMinutes(
		stages.TotalLightSleepTimeMilli +
			stages.TotalSlowWaveSleepTimeMilli +
			stages.TotalREMSleepTimeMilli,
	)

	sleepNeededMinutes := millisToMinutes(
		score.SleepNeeded.BaselineMilli +
			score.SleepNeeded.NeedFromSleepDebtMilli +
			score.SleepNeeded.NeedFromRecentStrainMilli +
			score.SleepNeeded.NeedFromRecentNapMilli,
	)

	snapshot.SleepScore = intPtr(roundToInt(score.SleepPerformancePercentage))
	snapshot.SleepMinutes = intPtr(sleepMinutes)
	snapshot.SleepNeededMinutes = intPtr(sleepNeededMinutes)
	snapshot.SleepVsNeedPct = intPtr(sleepVsNeedPct(sleepMinutes, sleepNeededMinutes))

	snapshot.AwakeMinutes = intPtr(millisToMinutes(stages.TotalAwakeTimeMilli))
	snapshot.LightSleepMinutes = intPtr(millisToMinutes(stages.TotalLightSleepTimeMilli))
	snapshot.DeepSleepMinutes = intPtr(millisToMinutes(stages.TotalSlowWaveSleepTimeMilli))
	snapshot.REMSleepMinutes = intPtr(millisToMinutes(stages.TotalREMSleepTimeMilli))
	snapshot.RestorativeMinutes = intPtr(millisToMinutes(stages.TotalSlowWaveSleepTimeMilli + stages.TotalREMSleepTimeMilli))

	snapshot.SleepEfficiencyPct = floatPtr(score.SleepEfficiencyPercentage)
	snapshot.SleepConsistencyPct = floatPtr(score.SleepConsistencyPercentage)
	snapshot.RespiratoryRate = floatPtr(score.RespiratoryRate)
}

func applyRecovery(snapshot *storage.DailyHealthSnapshot, recovery *whoopapi.RecoveryRecord) {
	if snapshot == nil || recovery == nil {
		return
	}

	snapshot.RecoveryWHOOPID = stringPtr(recoveryID(recovery.Recovery))

	if recovery.Recovery.ScoreState != whoopapi.ScoreStateScored || recovery.Recovery.Score == nil {
		return
	}

	score := recovery.Recovery.Score

	snapshot.RecoveryScore = intPtr(roundToInt(score.RecoveryScore))
	snapshot.HRVRMSSDMS = floatPtr(score.HRVRMSSDMilli)
	snapshot.RestingHeartRateBPM = intPtr(roundToInt(score.RestingHeartRate))
	snapshot.SpO2Pct = floatPtr(score.SpO2Percentage)
	snapshot.SkinTempCelsius = floatPtr(score.SkinTempCelsius)
}

func applyCycle(snapshot *storage.DailyHealthSnapshot, cycle *whoopapi.CycleRecord) {
	if snapshot == nil || cycle == nil {
		return
	}

	snapshot.CycleWHOOPID = stringPtr(strconv.FormatInt(cycle.Cycle.ID, 10))

	if cycle.Cycle.ScoreState != whoopapi.ScoreStateScored || cycle.Cycle.Score == nil {
		return
	}

	snapshot.DayStrain = floatPtr(cycle.Cycle.Score.Strain)
}

func detectDataState(snapshot storage.DailyHealthSnapshot, sleep *whoopapi.SleepRecord, recovery *whoopapi.RecoveryRecord, cycle *whoopapi.CycleRecord) storage.DataState {
	if sleep == nil && recovery == nil && cycle == nil {
		return storage.DataStatePending
	}

	if snapshot.SleepScore != nil && snapshot.RecoveryScore != nil && snapshot.DayStrain != nil {
		return storage.DataStateReady
	}

	return storage.DataStatePartial
}

func buildSleepRawObjects(userID int64, fetchedAt time.Time, records []whoopapi.SleepRecord) []storage.RawWHOOPObject {
	out := make([]storage.RawWHOOPObject, 0, len(records))

	for _, record := range records {
		if len(record.Raw) == 0 || strings.TrimSpace(record.Sleep.ID) == "" {
			continue
		}

		out = append(out, storage.RawWHOOPObject{
			UserID:      userID,
			ObjectType:  whoopapi.ObjectTypeSleep,
			WHOOPID:     record.Sleep.ID,
			StartAt:     timePtr(record.Sleep.Start),
			EndAt:       timePtr(record.Sleep.End),
			ScoreState:  scoreStatePtr(record.Sleep.ScoreState),
			PayloadJSON: json.RawMessage(record.Raw),
			FetchedAt:   fetchedAt,
		})
	}

	return out
}

func buildRecoveryRawObjects(userID int64, fetchedAt time.Time, records []whoopapi.RecoveryRecord) []storage.RawWHOOPObject {
	out := make([]storage.RawWHOOPObject, 0, len(records))

	for _, record := range records {
		if len(record.Raw) == 0 {
			continue
		}

		id := recoveryID(record.Recovery)
		if strings.TrimSpace(id) == "" {
			continue
		}

		out = append(out, storage.RawWHOOPObject{
			UserID:      userID,
			ObjectType:  whoopapi.ObjectTypeRecovery,
			WHOOPID:     id,
			ScoreState:  scoreStatePtr(record.Recovery.ScoreState),
			PayloadJSON: json.RawMessage(record.Raw),
			FetchedAt:   fetchedAt,
		})
	}

	return out
}

func buildCycleRawObjects(userID int64, fetchedAt time.Time, records []whoopapi.CycleRecord) []storage.RawWHOOPObject {
	out := make([]storage.RawWHOOPObject, 0, len(records))

	for _, record := range records {
		if len(record.Raw) == 0 || record.Cycle.ID == 0 {
			continue
		}

		out = append(out, storage.RawWHOOPObject{
			UserID:      userID,
			ObjectType:  whoopapi.ObjectTypeCycle,
			WHOOPID:     strconv.FormatInt(record.Cycle.ID, 10),
			StartAt:     timePtr(record.Cycle.Start),
			EndAt:       timePtr(record.Cycle.End),
			ScoreState:  scoreStatePtr(record.Cycle.ScoreState),
			PayloadJSON: json.RawMessage(record.Raw),
			FetchedAt:   fetchedAt,
		})
	}

	return out
}

func buildWorkoutRawObjects(userID int64, fetchedAt time.Time, records []whoopapi.WorkoutRecord) []storage.RawWHOOPObject {
	out := make([]storage.RawWHOOPObject, 0, len(records))

	for _, record := range records {
		if len(record.Raw) == 0 || strings.TrimSpace(record.Workout.ID) == "" {
			continue
		}

		out = append(out, storage.RawWHOOPObject{
			UserID:      userID,
			ObjectType:  whoopapi.ObjectTypeWorkout,
			WHOOPID:     record.Workout.ID,
			StartAt:     timePtr(record.Workout.Start),
			EndAt:       timePtr(record.Workout.End),
			ScoreState:  scoreStatePtr(record.Workout.ScoreState),
			PayloadJSON: json.RawMessage(record.Raw),
			FetchedAt:   fetchedAt,
		})
	}

	return out
}

func recoveryID(recovery whoopapi.Recovery) string {
	if strings.TrimSpace(recovery.SleepID) != "" {
		return recovery.SleepID
	}
	if recovery.CycleID != 0 {
		return strconv.FormatInt(recovery.CycleID, 10)
	}

	return ""
}

func normalizeSnapshotDate(date time.Time, now time.Time, loc *time.Location, sleep *whoopapi.SleepRecord) time.Time {
	if !date.IsZero() {
		y, m, d := date.In(loc).Date()
		return dateOnlyUTC(y, m, d)
	}

	if sleep != nil && !sleep.Sleep.End.IsZero() {
		y, m, d := sleep.Sleep.End.In(loc).Date()
		return dateOnlyUTC(y, m, d)
	}

	y, m, d := now.In(loc).Date()
	return dateOnlyUTC(y, m, d)
}

func dateOnlyUTC(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

func loadLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultTimezone
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
}

func millisToMinutes(ms int64) int {
	if ms <= 0 {
		return 0
	}

	return int(math.Round(float64(ms) / float64(time.Minute/time.Millisecond)))
}

func sleepVsNeedPct(sleepMinutes int, neededMinutes int) int {
	if sleepMinutes <= 0 || neededMinutes <= 0 {
		return 0
	}

	return int(math.Round(float64(sleepMinutes) / float64(neededMinutes) * 100))
}

func latestUpdatedAt(values ...time.Time) *time.Time {
	var latest time.Time

	for _, value := range values {
		if value.IsZero() {
			continue
		}
		if latest.IsZero() || value.After(latest) {
			latest = value
		}
	}

	if latest.IsZero() {
		return nil
	}

	return &latest
}

func sleepUpdatedAt(record *whoopapi.SleepRecord) time.Time {
	if record == nil {
		return time.Time{}
	}

	return record.Sleep.UpdatedAt
}

func recoveryUpdatedAt(record *whoopapi.RecoveryRecord) time.Time {
	if record == nil {
		return time.Time{}
	}

	return record.Recovery.UpdatedAt
}

func cycleUpdatedAt(record *whoopapi.CycleRecord) time.Time {
	if record == nil {
		return time.Time{}
	}

	return record.Cycle.UpdatedAt
}

func stringPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}

	return &v
}

func intPtr(v int) *int {
	return &v
}

func floatPtr(v float64) *float64 {
	return &v
}

func timePtr(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}

	v = v.UTC()
	return &v
}

func scoreStatePtr(v whoopapi.ScoreState) *whoopapi.ScoreState {
	if strings.TrimSpace(string(v)) == "" {
		return nil
	}

	return &v
}

func roundToInt(v float64) int {
	return int(math.Round(v))
}
