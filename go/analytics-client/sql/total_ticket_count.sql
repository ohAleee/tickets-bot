SELECT uniqExactMerge(count) AS total_count
FROM analytics.total_ticket_count
WHERE guild_id = ?;