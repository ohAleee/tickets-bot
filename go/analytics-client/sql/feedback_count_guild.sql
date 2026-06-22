SELECT countMerge(count) AS total_count
FROM analytics.feedback_count_guild
WHERE guild_id = ?;