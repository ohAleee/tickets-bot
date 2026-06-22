use super::{Component, ComponentType};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct Container {
    pub r#type: ComponentType,
    pub components: Vec<Component>,
}
