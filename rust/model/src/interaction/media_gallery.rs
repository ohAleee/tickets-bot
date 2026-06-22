use super::{ComponentType};
use serde::{Deserialize, Serialize};
use super::thumbnail::UnfurledMediaItem;

#[derive(Serialize, Deserialize, Debug)]
pub struct MediaGallery {
    pub r#type: ComponentType,
    pub items: Vec<UnfurledMediaItem>,
}