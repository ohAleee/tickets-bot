use super::{ComponentType};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct Thumbnail {
    pub r#type: ComponentType,
    pub media: UnfurledMediaItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<Box<str>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub spoiler: Option<bool>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct UnfurledMediaItem {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub url: Option<Box<str>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub proxy_url: Option<Box<str>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content_type: Option<Box<str>>,
}

