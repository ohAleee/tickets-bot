package analytics

import "context"

func (c *Client) GetGuildCount(context context.Context, integrationId int) (int, error) {
	query := `
SELECT countMerge(count)
FROM analytics.custom_integration_guild_counts
WHERE integration_id=?
GROUP BY integration_id`

	var count int
	if err := c.client.QueryRow(context, query, integrationId).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
