#![no_std]

use soroban_sdk::{
    contract, contracterror, contractimpl, contracttype, panic_with_error, symbol_short, Address,
    Env, Symbol,
};

#[contract]
pub struct YieldRegistryContract;

#[derive(Clone, Debug, Eq, PartialEq)]
#[contracttype]
pub enum SourceStatus {
    Active,
    Paused,
    Disabled,
}

#[contracttype]
enum DataKey {
    Admin,
    Source(Symbol),
}

#[contracterror]
#[derive(Copy, Clone, Debug, Eq, PartialEq, PartialOrd, Ord)]
#[repr(u32)]
pub enum YieldRegistryError {
    AlreadyInitialized = 1,
    Unauthorized = 2,
    UnknownSource = 3,
}

#[contractimpl]
impl YieldRegistryContract {
    pub fn initialize(env: Env, admin: Address) {
        if env.storage().instance().has(&DataKey::Admin) {
            panic_with_error!(&env, YieldRegistryError::AlreadyInitialized);
        }

        admin.require_auth();
        env.storage().instance().set(&DataKey::Admin, &admin);
    }

    pub fn upsert_source(env: Env, admin: Address, source_id: Symbol, status: SourceStatus) {
        admin.require_auth();
        require_admin(&env, &admin);
        env.storage()
            .persistent()
            .set(&DataKey::Source(source_id.clone()), &status);
        env.events()
            .publish((symbol_short!("source"), source_id), status);
    }

    pub fn has_source(env: Env, source_id: Symbol) -> bool {
        env.storage().persistent().has(&DataKey::Source(source_id))
    }

    pub fn get_source_status(env: Env, source_id: Symbol) -> SourceStatus {
        env.storage()
            .persistent()
            .get(&DataKey::Source(source_id))
            .unwrap_or_else(|| panic_with_error!(&env, YieldRegistryError::UnknownSource))
    }
}

fn require_admin(env: &Env, admin: &Address) {
    let stored_admin: Address = env
        .storage()
        .instance()
        .get(&DataKey::Admin)
        .unwrap_or_else(|| panic_with_error!(env, YieldRegistryError::Unauthorized));

    if stored_admin != *admin {
        panic_with_error!(env, YieldRegistryError::Unauthorized);
    }
}

mod test;
