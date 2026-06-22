use crate::patreon::tier::Tier::{Premium, Whitelabel};
use serde::{Serialize, Serializer};
use tracing::log::{warn};
#[derive(Debug, Copy, Clone, PartialEq, Eq)]
pub enum Tier {
    Premium,
    Whitelabel,
}

// TODO: Don't store these as constants
pub const TIERS_PREMIUM_LEGACY: &[usize] = &[];
pub const TIERS_WHITELABEL_LEGACY: &[usize] = &[24807722];
pub const TIERS_PREMIUM: &[usize] = &[24807710, 24860764];
pub const TIERS_WHITELABEL: &[usize] = &[];

// TODO: Don't store these as constants
pub const TIER_PRIORITY: &[(usize, i32)] = &[
    (24807722, 3),
    (24860764, 2),
    (24807710, 1),
    (24807661, 0),
];

impl Tier {
    pub fn get_priority(&self, patreon_id: usize) -> i32 {
        TIER_PRIORITY
            .iter()
            .find(|(id, _)| *id == patreon_id)
            .map(|(_, priority)| *priority)
            .unwrap_or_else(|| {
                // If the tier is not in TIER_PRIORITY, set the priority to 4 (highest) as it's probably from a key or another source
                warn!("Tier {:?} not found in TIER_PRIORITY", self);
                4
            })
    }

    pub fn get_by_patreon_id(patreon_id: usize) -> Option<Tier> {
        if TIERS_WHITELABEL_LEGACY.contains(&patreon_id) || TIERS_WHITELABEL.contains(&patreon_id) {
            Some(Tier::Whitelabel)
        } else if TIERS_PREMIUM_LEGACY.contains(&patreon_id) || TIERS_PREMIUM.contains(&patreon_id)
        {
            Some(Tier::Premium)
        } else {
            None
        }
    }

    pub fn tier_id(&self) -> i32 {
        match self {
            Tier::Premium => 0,
            Tier::Whitelabel => 1,
        }
    }

    pub fn values<'a>() -> &'a [Tier] {
        &[Premium, Whitelabel]
    }

    pub fn sku_label(&self) -> &'static str {
        match self {
            Tier::Premium => "premium",
            Tier::Whitelabel => "whitelabel",
        }
    }

    pub fn inherited_tiers(&self) -> Vec<Tier> {
        match self {
            Tier::Premium => vec![],
            Tier::Whitelabel => vec![Tier::Premium],
        }
    }
}

impl Serialize for Tier {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_i32(self.tier_id())
    }
}
