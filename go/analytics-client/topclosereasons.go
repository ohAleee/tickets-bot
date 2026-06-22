package analytics

import (
	"context"
)

func (c *Client) GetTopCloseReasons(context context.Context, guildId uint64, panelId *int) ([]string, error) {
	query := `
SELECT close_reason
FROM analytics.top_close_reasons
WHERE guild_id = ? AND panel_id = ?
ORDER BY ranking ASC
LIMIT 10`

	rows, err := c.client.Query(context, query, guildId, panelId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	reasons := make([]string, 10)
	i := 0
	for rows.Next() {
		var reason string
		if err := rows.Scan(&reason); err != nil {
			return nil, err
		}

		reasons[i] = reason
		i++
	}

	return reasons[:i], nil
}

func (c *Client) GetTopCloseReasonsWithPrefix(context context.Context, guildId uint64, panelId *int, prefix string) ([]string, error) {
	query := `
SELECT close_reason
FROM analytics.top_close_reasons
WHERE guild_id = ? AND panel_id = ? AND LOWER(close_reason) LIKE LOWER(?) || '%'
ORDER BY ranking ASC
LIMIT 10`

	rows, err := c.client.Query(context, query, guildId, panelId, prefix)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	reasons := make([]string, 10)
	i := 0
	for rows.Next() {
		var reason string
		if err := rows.Scan(&reason); err != nil {
			return nil, err
		}

		reasons[i] = reason
		i++
	}

	return reasons[:i], nil
}

func (c *Client) GetTopCloseReasonsContaining(context context.Context, guildId uint64, panelId *int, contains string) ([]string, error) {
	query := `
SELECT close_reason
FROM analytics.top_close_reasons
WHERE guild_id = $1 AND panel_id = $2 AND LOWER(close_reason) LIKE '%' || LOWER($3) || '%'
ORDER BY CASE WHEN LOWER(close_reason) LIKE LOWER($3) || '%' THEN 0 ELSE 1 END,
ranking ASC
LIMIT 10`

	rows, err := c.client.Query(context, query, guildId, panelId, contains)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	reasons := make([]string, 10)
	i := 0
	for rows.Next() {
		var reason string
		if err := rows.Scan(&reason); err != nil {
			return nil, err
		}

		reasons[i] = reason
		i++
	}

	return reasons[:i], nil
}
