use async_trait::async_trait;

use sqlx::{postgres::PgPool, Error};
use std::sync::Arc;

use crate::Table;
use model::Snowflake;

/// Which bot the owner chose to serve a guild. Kept separate from whitelabel_guilds (observed
/// membership) so that guild joins and guild syncs cannot clobber the choice.
pub struct WhitelabelGuildAssignments {
    db: Arc<PgPool>,
}

#[async_trait]
impl Table for WhitelabelGuildAssignments {
    async fn create_schema(&self) -> Result<(), Error> {
        sqlx::query(
            r#"
CREATE TABLE IF NOT EXISTS whitelabel_guild_assignments(
	"guild_id" int8 NOT NULL,
	"bot_id" int8 NOT NULL,
	"assigned_by" int8,
	"assigned_at" timestamptz NOT NULL DEFAULT NOW(),
	FOREIGN KEY("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("guild_id")
);
CREATE INDEX IF NOT EXISTS whitelabel_guild_assignments_bot_id ON whitelabel_guild_assignments("bot_id");
"#,
        )
        .execute(&*self.db)
        .await?;

        Ok(())
    }
}

impl WhitelabelGuildAssignments {
    pub fn new(db: Arc<PgPool>) -> WhitelabelGuildAssignments {
        WhitelabelGuildAssignments { db }
    }

    pub async fn get(&self, guild_id: Snowflake) -> Result<Option<Snowflake>, Error> {
        let query = r#"SELECT "bot_id" FROM whitelabel_guild_assignments WHERE "guild_id" = $1;"#;

        match sqlx::query_as::<_, (i64,)>(query)
            .bind(guild_id.0 as i64)
            .fetch_one(&*self.db)
            .await
        {
            Ok(id) => Ok(Some(Snowflake(id.0 as u64))),
            Err(sqlx::Error::RowNotFound) => Ok(None),
            Err(e) => Err(e),
        }
    }

    /// First bot to join a guild serves it; an existing choice is never overwritten.
    pub async fn insert_if_absent(
        &self,
        guild_id: Snowflake,
        bot_id: Snowflake,
    ) -> Result<(), Error> {
        let query = r#"
INSERT INTO whitelabel_guild_assignments("guild_id", "bot_id")
VALUES($1, $2)
ON CONFLICT("guild_id") DO NOTHING;"#;

        sqlx::query(query)
            .bind(guild_id.0 as i64)
            .bind(bot_id.0 as i64)
            .execute(&*self.db)
            .await?;

        Ok(())
    }

    pub async fn delete_if_bot(&self, guild_id: Snowflake, bot_id: Snowflake) -> Result<(), Error> {
        let query =
            r#"DELETE FROM whitelabel_guild_assignments WHERE "guild_id" = $1 AND "bot_id" = $2;"#;

        sqlx::query(query)
            .bind(guild_id.0 as i64)
            .bind(bot_id.0 as i64)
            .execute(&*self.db)
            .await?;

        Ok(())
    }
}
