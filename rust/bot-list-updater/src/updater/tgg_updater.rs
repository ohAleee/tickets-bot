use super::Updater;
use crate::UpdaterError;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};

pub struct TggUpdater {
    token: String,
    bot_id: u64,
    http_client: reqwest::Client,
}

impl TggUpdater {
    pub fn new(token: String, bot_id: u64) -> TggUpdater {
        TggUpdater {
            token,
            bot_id,
            http_client: reqwest::Client::new(),
        }
    }

    pub fn new_with_client(token: String, bot_id: u64, http_client: reqwest::Client) -> TggUpdater {
        TggUpdater {
            token,
            bot_id,
            http_client,
        }
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TggRequest {
    pub server_count: usize,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub shard_id: Option<u16>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TggResponse {
    pub message: String,
}

#[async_trait]
impl Updater for TggUpdater {
    async fn update(&self, count: usize) -> Result<(), UpdaterError> {
        let url = format!(
            "https://top.gg/api/bots/{}/stats",
            self.bot_id
        );

        let body = TggRequest {
            server_count: count,
            shard_id: None,
        };

        let res = self
            .http_client
            .post(url)
            .header("Authorization", &self.token[..])
            .json(&body)
            .send()
            .await
            .map_err(UpdaterError::ReqwestError)?;

        if res.status().is_success() {
            Ok(())
        } else {
            let body: TggResponse = res.json().await.map_err(UpdaterError::ReqwestError)?;
            UpdaterError::ResponseError(body.message).into()
        }
    }
}
