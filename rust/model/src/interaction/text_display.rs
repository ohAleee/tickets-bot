use super::{ComponentType};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct TextDisplay {
    pub r#type: ComponentType,
    pub content: Box<str>,
}