#[cfg(feature = "whitelabel")]
use crate::gateway::event_forwarding::EventForwarder;
#[cfg(feature = "whitelabel")]
use crate::{GatewayError, Result, Shard};
#[cfg(feature = "whitelabel")]
use model::Snowflake;

#[cfg(feature = "whitelabel")]
impl<T: EventForwarder> Shard<T> {
    // Shard::user_id holds the *bot* id for whitelabel shards.
    pub async fn store_whitelabel_guild(&self, guild_id: Snowflake) -> Result<()> {
        self.database
            .whitelabel_guilds
            .insert(self.user_id, guild_id)
            .await
            .map_err(GatewayError::DatabaseError)?;

        // First bot to join a guild serves it. An owner who already picked a bot for this guild
        // keeps it: insert_if_absent never overwrites an existing assignment.
        self.database
            .whitelabel_guild_assignments
            .insert_if_absent(guild_id, self.user_id)
            .await
            .map_err(GatewayError::DatabaseError)
    }
}

pub fn is_whitelabel() -> bool {
    cfg!(feature = "whitelabel")
}
