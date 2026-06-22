package analytics

import (
	"context"
	_ "embed"
)

var (
	//go:embed sql/average_rating_guild.sql
	queryGetAverageFeedbackRatingGuild string

	//go:embed sql/feedback_count_guild.sql
	queryGetFeedbackCountGuild string
)

func (c *Client) GetAverageFeedbackRatingGuild(ctx context.Context, guildId uint64) (float64, error) {
	var rating float64
	if err := c.client.QueryRow(ctx, queryGetAverageFeedbackRatingGuild, guildId).Scan(&rating); err != nil {
		return 0, err
	}

	return rating, nil
}

func (c *Client) GetFeedbackCountGuild(ctx context.Context, guildId uint64) (uint64, error) {
	var count uint64
	if err := c.client.QueryRow(ctx, queryGetFeedbackCountGuild, guildId).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
