SELECT avgOrNullMerge(rating) AS average_rating
FROM analytics.average_rating_guild
WHERE guild_id = ?;