package whoopapi

import (
	"encoding/json"
	"time"
)

type ScoreState string

const (
	ScoreStateScored       ScoreState = "SCORED"
	ScoreStatePendingScore ScoreState = "PENDING_SCORE"
	ScoreStateUnscorable   ScoreState = "UNSCORABLE"
)

type ObjectType string

const (
	ObjectTypeCycle    ObjectType = "cycle"
	ObjectTypeSleep    ObjectType = "sleep"
	ObjectTypeRecovery ObjectType = "recovery"
	ObjectTypeWorkout  ObjectType = "workout"
)

type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"-"`
}

type CollectionResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token"`
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return e.Body
}

type Cycle struct {
	ID             int64           `json:"id"`
	UserID         int64           `json:"user_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Start          time.Time       `json:"start"`
	End            time.Time       `json:"end"`
	TimezoneOffset string          `json:"timezone_offset"`
	ScoreState     ScoreState      `json:"score_state"`
	Score          *CycleScore     `json:"score"`
	Raw            json.RawMessage `json:"-"`
}

type CycleScore struct {
	Strain           float64 `json:"strain"`
	Kilojoule        float64 `json:"kilojoule"`
	AverageHeartRate int     `json:"average_heart_rate"`
	MaxHeartRate     int     `json:"max_heart_rate"`
}

type Sleep struct {
	ID             string          `json:"id"`
	CycleID        int64           `json:"cycle_id"`
	V1ID           int64           `json:"v1_id"`
	UserID         int64           `json:"user_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Start          time.Time       `json:"start"`
	End            time.Time       `json:"end"`
	TimezoneOffset string          `json:"timezone_offset"`
	Nap            bool            `json:"nap"`
	ScoreState     ScoreState      `json:"score_state"`
	Score          *SleepScore     `json:"score"`
	Raw            json.RawMessage `json:"-"`
}

type SleepScore struct {
	StageSummary               SleepStageSummary `json:"stage_summary"`
	SleepNeeded                SleepNeeded       `json:"sleep_needed"`
	RespiratoryRate            float64           `json:"respiratory_rate"`
	SleepPerformancePercentage float64           `json:"sleep_performance_percentage"`
	SleepConsistencyPercentage float64           `json:"sleep_consistency_percentage"`
	SleepEfficiencyPercentage  float64           `json:"sleep_efficiency_percentage"`
}

type SleepStageSummary struct {
	TotalInBedTimeMilli         int64 `json:"total_in_bed_time_milli"`
	TotalAwakeTimeMilli         int64 `json:"total_awake_time_milli"`
	TotalNoDataTimeMilli        int64 `json:"total_no_data_time_milli"`
	TotalLightSleepTimeMilli    int64 `json:"total_light_sleep_time_milli"`
	TotalSlowWaveSleepTimeMilli int64 `json:"total_slow_wave_sleep_time_milli"`
	TotalREMSleepTimeMilli      int64 `json:"total_rem_sleep_time_milli"`
	SleepCycleCount             int   `json:"sleep_cycle_count"`
	DisturbanceCount            int   `json:"disturbance_count"`
}

type SleepNeeded struct {
	BaselineMilli             int64 `json:"baseline_milli"`
	NeedFromSleepDebtMilli    int64 `json:"need_from_sleep_debt_milli"`
	NeedFromRecentStrainMilli int64 `json:"need_from_recent_strain_milli"`
	NeedFromRecentNapMilli    int64 `json:"need_from_recent_nap_milli"`
}

type Recovery struct {
	CycleID    int64           `json:"cycle_id"`
	SleepID    string          `json:"sleep_id"`
	UserID     int64           `json:"user_id"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ScoreState ScoreState      `json:"score_state"`
	Score      *RecoveryScore  `json:"score"`
	Raw        json.RawMessage `json:"-"`
}

type RecoveryScore struct {
	UserCalibrating  bool    `json:"user_calibrating"`
	RecoveryScore    float64 `json:"recovery_score"`
	RestingHeartRate float64 `json:"resting_heart_rate"`
	HRVRMSSDMilli    float64 `json:"hrv_rmssd_milli"`
	SpO2Percentage   float64 `json:"spo2_percentage"`
	SkinTempCelsius  float64 `json:"skin_temp_celsius"`
}

type Workout struct {
	ID             string          `json:"id"`
	V1ID           int64           `json:"v1_id"`
	UserID         int64           `json:"user_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Start          time.Time       `json:"start"`
	End            time.Time       `json:"end"`
	TimezoneOffset string          `json:"timezone_offset"`
	SportName      string          `json:"sport_name"`
	SportID        int             `json:"sport_id"`
	ScoreState     ScoreState      `json:"score_state"`
	Score          *WorkoutScore   `json:"score"`
	Raw            json.RawMessage `json:"-"`
}

type WorkoutScore struct {
	Strain              float64       `json:"strain"`
	AverageHeartRate    int           `json:"average_heart_rate"`
	MaxHeartRate        int           `json:"max_heart_rate"`
	Kilojoule           float64       `json:"kilojoule"`
	PercentRecorded     float64       `json:"percent_recorded"`
	DistanceMeter       float64       `json:"distance_meter"`
	AltitudeGainMeter   float64       `json:"altitude_gain_meter"`
	AltitudeChangeMeter float64       `json:"altitude_change_meter"`
	ZoneDurations       ZoneDurations `json:"zone_durations"`
}

type ZoneDurations struct {
	ZoneZeroMilli  int64 `json:"zone_zero_milli"`
	ZoneOneMilli   int64 `json:"zone_one_milli"`
	ZoneTwoMilli   int64 `json:"zone_two_milli"`
	ZoneThreeMilli int64 `json:"zone_three_milli"`
	ZoneFourMilli  int64 `json:"zone_four_milli"`
	ZoneFiveMilli  int64 `json:"zone_five_milli"`
}
