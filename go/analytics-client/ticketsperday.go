package analytics

import (
	"context"
	_ "embed"
	"time"
)

type CountOnDate struct {
	Date  time.Time `json:"date"`
	Count uint64    `json:"count"`
}

var (
	//go:embed sql/last_n_tickets_per_day_guild.sql
	queryLastNTicketsPerDayGuildWide string
)

func (c *Client) GetLastNTicketsPerDayGuild(ctx context.Context, guildId uint64, nDays int) ([]CountOnDate, error) {
	rows, err := c.client.Query(ctx, queryLastNTicketsPerDayGuildWide, guildId, nDays)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	counts := make([]CountOnDate, 0, nDays)
	for rows.Next() {
		var count CountOnDate
		if err := rows.Scan(&count.Date, &count.Count); err != nil {
			return nil, err
		}

		counts = append(counts, count)
	}

	return counts, nil
}
