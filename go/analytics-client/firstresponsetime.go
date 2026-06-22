package analytics

import (
	"context"
	"time"
)

func (c *Client) GetFirstResponseTimeStats(context context.Context, guildId uint64) (TripleWindow, error) {
	query := `
SELECT
    avgMerge(all_time),
    avgOrNullMerge(monthly),
    avgOrNullMerge(weekly)
FROM analytics.first_response_time_guild
WHERE guild_id = ?
GROUP BY guild_id`

	rows, err := c.client.Query(context, query, guildId)
	if err != nil {
		return TripleWindow{}, err
	}

	defer rows.Close()
	if rows.Next() {
		// Values in seconds
		var allTime float64
		var monthly, weekly *float64
		if err := rows.Scan(&allTime, &monthly, &weekly); err != nil {
			return TripleWindow{}, nil
		}

		return TripleWindow{
			AllTime: ptr(time.Duration(allTime) * time.Second),
			Monthly: mapNullableSecondsToDuration(monthly),
			Weekly:  mapNullableSecondsToDuration(weekly),
		}, nil
	} else {
		return blankTripleWindow(), nil
	}
}
