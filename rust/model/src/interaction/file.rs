use super::{ComponentType};
use serde::{Deserialize, Serialize};
use super::thumbnail::UnfurledMediaItem;

#[derive(Serialize, Deserialize, Debug)]
pub struct File {
    pub r#type: ComponentType,
    pub file: UnfurledMediaItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub spoiler: Option<bool>,
}