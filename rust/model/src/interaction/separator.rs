use super::{ComponentType};
use serde::{Deserialize, Serialize};
use super::thumbnail::UnfurledMediaItem;

#[derive(Serialize, Deserialize, Debug)]
pub struct Separator {
    pub r#type: ComponentType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub divider: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub spacer: Option<u32>,
}