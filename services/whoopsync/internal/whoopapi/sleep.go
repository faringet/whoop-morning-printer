package whoopapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const sleepCollectionEndpoint = "/activity/sleep"

type SleepQuery struct {
	Start time.Time
	End   time.Time
	Limit int
}

type SleepRecord struct {
	Sleep Sleep
	Raw   json.RawMessage
}

func (c *Client) GetSleeps(ctx context.Context, accessToken string, query SleepQuery) ([]SleepRecord, error) {
	values := collectionQueryValues(query.Start, query.End, query.Limit)

	rawRecords, err := c.GetCollection(ctx, accessToken, sleepCollectionEndpoint, values)
	if err != nil {
		return nil, fmt.Errorf("whoop api: get sleeps: %w", err)
	}

	out := make([]SleepRecord, 0, len(rawRecords))

	for _, raw := range rawRecords {
		var sleep Sleep
		if err := c.Decode(raw, &sleep); err != nil {
			return nil, fmt.Errorf("whoop api: decode sleep: %w", err)
		}

		sleep.Raw = raw

		out = append(out, SleepRecord{
			Sleep: sleep,
			Raw:   raw,
		})
	}

	return out, nil
}

func collectionQueryValues(start time.Time, end time.Time, limit int) url.Values {
	values := url.Values{}

	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}
	if !start.IsZero() {
		values.Set("start", formatWHOOPTime(start))
	}
	if !end.IsZero() {
		values.Set("end", formatWHOOPTime(end))
	}

	return values
}

func formatWHOOPTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
