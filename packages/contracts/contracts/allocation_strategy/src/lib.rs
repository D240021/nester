#![no_std]

use soroban_sdk::{
    contract, contractclient, contracterror, contractimpl, contracttype, panic_with_error,
    symbol_short, Address, Env, Symbol, Vec,
};

#[contract]
pub struct AllocationStrategyContract;

#[derive(Clone, Debug, Eq, PartialEq)]
#[contracttype]
pub struct AllocationWeight {
    pub source_id: Symbol,
    pub weight_bps: u32,
}

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
    RegistryAddress,
    Weights,
}

#[contracterror]
#[derive(Copy, Clone, Debug, Eq, PartialEq, PartialOrd, Ord)]
#[repr(u32)]
pub enum AllocationStrategyError {
    AlreadyInitialized = 1,
    Unauthorized = 2,
    InvalidWeightSum = 3,
    UnknownSource = 4,
    InactiveSource = 5,
    EmptyWeights = 6,
    NegativeAmount = 7,
    DuplicateSource = 8,
}

#[contractclient(name = "YieldRegistryClient")]
pub trait YieldRegistryContract {
    fn has_source(env: Env, source_id: Symbol) -> bool;
    fn get_source_status(env: Env, source_id: Symbol) -> SourceStatus;
}

#[contractimpl]
impl AllocationStrategyContract {
    pub fn initialize(env: Env, admin: Address, registry_address: Address) {
        if env.storage().instance().has(&DataKey::Admin) {
            panic_with_error!(&env, AllocationStrategyError::AlreadyInitialized);
        }

        admin.require_auth();
        env.storage().instance().set(&DataKey::Admin, &admin);
        env.storage()
            .instance()
            .set(&DataKey::RegistryAddress, &registry_address);
        env.storage()
            .instance()
            .set(&DataKey::Weights, &Vec::<AllocationWeight>::new(&env));
    }

    pub fn set_weights(env: Env, admin: Address, weights: Vec<AllocationWeight>) {
        admin.require_auth();
        require_admin(&env, &admin);
        validate_weights(&env, &weights);

        env.storage().instance().set(&DataKey::Weights, &weights);
        env.events()
            .publish((symbol_short!("weights"),), weights.clone());
    }

    pub fn get_weights(env: Env) -> Vec<AllocationWeight> {
        read_weights(&env)
    }

    pub fn calculate_allocation(env: Env, total_amount: i128) -> Vec<(Symbol, i128)> {
        if total_amount < 0 {
            panic_with_error!(&env, AllocationStrategyError::NegativeAmount);
        }

        let weights = read_weights(&env);
        let mut allocations: Vec<(Symbol, i128)> = Vec::new(&env);
        if weights.is_empty() {
            return allocations;
        }

        let mut distributed = 0_i128;
        let mut highest_weight = 0_u32;
        let mut highest_index = 0_u32;

        for index in 0..weights.len() {
            let weight = weights.get_unchecked(index);
            let amount = total_amount * weight.weight_bps as i128 / 10_000_i128;
            allocations.push_back((weight.source_id.clone(), amount));
            distributed += amount;

            if weight.weight_bps > highest_weight {
                highest_weight = weight.weight_bps;
                highest_index = index;
            }
        }

        let remainder = total_amount - distributed;
        if remainder != 0 {
            let (source_id, current_amount) = allocations.get_unchecked(highest_index);
            allocations.set(highest_index, (source_id, current_amount + remainder));
        }

        allocations
    }

    pub fn get_source_allocation(env: Env, source_id: Symbol) -> u32 {
        let weights = read_weights(&env);
        for weight in weights.iter() {
            if weight.source_id == source_id {
                return weight.weight_bps;
            }
        }

        0
    }
}

fn require_admin(env: &Env, admin: &Address) {
    let stored_admin: Address = env
        .storage()
        .instance()
        .get(&DataKey::Admin)
        .unwrap_or_else(|| panic_with_error!(env, AllocationStrategyError::Unauthorized));

    if stored_admin != *admin {
        panic_with_error!(env, AllocationStrategyError::Unauthorized);
    }
}

fn read_weights(env: &Env) -> Vec<AllocationWeight> {
    env.storage()
        .instance()
        .get(&DataKey::Weights)
        .unwrap_or_else(|| Vec::new(env))
}

fn registry_client(env: &Env) -> YieldRegistryClient {
    let registry_address: Address = env
        .storage()
        .instance()
        .get(&DataKey::RegistryAddress)
        .unwrap_or_else(|| panic_with_error!(env, AllocationStrategyError::Unauthorized));
    YieldRegistryClient::new(env, &registry_address)
}

fn validate_weights(env: &Env, weights: &Vec<AllocationWeight>) {
    if weights.is_empty() {
        panic_with_error!(env, AllocationStrategyError::EmptyWeights);
    }

    let registry = registry_client(env);
    let mut sum = 0_u32;

    for index in 0..weights.len() {
        let weight = weights.get_unchecked(index);
        sum += weight.weight_bps;

        if is_duplicate_source(weights, index, &weight.source_id) {
            panic_with_error!(env, AllocationStrategyError::DuplicateSource);
        }

        if !registry.has_source(&weight.source_id) {
            panic_with_error!(env, AllocationStrategyError::UnknownSource);
        }

        if registry.get_source_status(&weight.source_id) != SourceStatus::Active {
            panic_with_error!(env, AllocationStrategyError::InactiveSource);
        }
    }

    if sum != 10_000 {
        panic_with_error!(env, AllocationStrategyError::InvalidWeightSum);
    }
}

fn is_duplicate_source(
    weights: &Vec<AllocationWeight>,
    current_index: u32,
    source_id: &Symbol,
) -> bool {
    for index in 0..current_index {
        if weights.get_unchecked(index).source_id == *source_id {
            return true;
        }
    }
    false
}

mod test;
