package analytics

import (
	"context"
	_ "embed"
)

var (
	//go:embed sql/total_ticket_count.sql
	queryGetTotalTicketCount string

	//go:embed sql/total_open_ticket_count.sql
	queryGetTotalOpenTicketCount string
)

func (c *Client) GetTotalTicketCount(ctx context.Context, guildId uint64) (uint64, error) {
	var count uint64
	if err := c.client.QueryRow(ctx, queryGetTotalTicketCount, guildId).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Client) GetTotalOpenTicketCount(ctx context.Context, guildId uint64) (uint64, error) {
	var count uint64
	if err := c.client.QueryRow(ctx, queryGetTotalOpenTicketCount, guildId).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
