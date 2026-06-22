use super::{Component, ComponentType};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct Label {
    pub r#type: ComponentType,
    pub label: Option<String>,
    pub description: Option<Box<str>>,
    pub component: Component,
}
