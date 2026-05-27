package whoopapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const cycleCollectionEndpoint = "/cycle"

type CycleQuery struct {
	Start time.Time
	End   time.Time
	Limit int
}

type CycleRecord struct {
	Cycle Cycle
	Raw   json.RawMessage
}

func (c *Client) GetCycles(ctx context.Context, accessToken string, query CycleQuery) ([]CycleRecord, error) {
	values := collectionQueryValues(query.Start, query.End, query.Limit)

	rawRecords, err := c.GetCollection(ctx, accessToken, cycleCollectionEndpoint, values)
	if err != nil {
		return nil, fmt.Errorf("whoop api: get cycles: %w", err)
	}

	out := make([]CycleRecord, 0, len(rawRecords))

	for _, raw := range rawRecords {
		var cycle Cycle
		if err := c.Decode(raw, &cycle); err != nil {
			return nil, fmt.Errorf("whoop api: decode cycle: %w", err)
		}

		cycle.Raw = raw

		out = append(out, CycleRecord{
			Cycle: cycle,
			Raw:   raw,
		})
	}

	return out, nil
}
