package whoopapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const recoveryCollectionEndpoint = "/recovery"

type RecoveryQuery struct {
	Start time.Time
	End   time.Time
	Limit int
}

type RecoveryRecord struct {
	Recovery Recovery
	Raw      json.RawMessage
}

func (c *Client) GetRecoveries(ctx context.Context, accessToken string, query RecoveryQuery) ([]RecoveryRecord, error) {
	values := collectionQueryValues(query.Start, query.End, query.Limit)

	rawRecords, err := c.GetCollection(ctx, accessToken, recoveryCollectionEndpoint, values)
	if err != nil {
		return nil, fmt.Errorf("whoop api: get recoveries: %w", err)
	}

	out := make([]RecoveryRecord, 0, len(rawRecords))

	for _, raw := range rawRecords {
		var recovery Recovery
		if err := c.Decode(raw, &recovery); err != nil {
			return nil, fmt.Errorf("whoop api: decode recovery: %w", err)
		}

		recovery.Raw = raw

		out = append(out, RecoveryRecord{
			Recovery: recovery,
			Raw:      raw,
		})
	}

	return out, nil
}
