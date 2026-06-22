use super::{Component, ComponentType};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct Section {
    pub r#type: ComponentType,
    pub components: Vec<Component>,
    pub accessory: Option<Component>,
}
