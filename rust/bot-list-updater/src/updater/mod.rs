use crate::UpdaterError;
use async_trait::async_trait;

mod dbl_updater;
mod tgg_updater;
pub use dbl_updater::DblUpdater;
pub use tgg_updater::TggUpdater;

#[async_trait]
pub trait Updater {
    async fn update(&self, count: usize) -> Result<(), UpdaterError>;
}
