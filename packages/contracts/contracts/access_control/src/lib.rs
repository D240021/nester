//! Nester shared access-control module (rlib — no on-chain state of its own).
//!
//! Reads/writes the *calling* contract's instance storage.
//!
//! # Roles
//! * [`Role::Admin`]    – full control; can grant/revoke any role.
//! * [`Role::Operator`] – day-to-day operations (e.g. weight updates).
//!
//! # Admin transfer (two-step)
//! 1. [`AccessControl::transfer_admin`] — stores a pending proposal.
//! 2. [`AccessControl::accept_admin`] — atomically grants new Admin, revokes old.
//!
//! # Last-admin protection
//! [`AccessControl::revoke_role`] panics with [`ContractError::InvalidOperation`]
//! when removing the last Admin.

#![no_std]

use soroban_sdk::{contracttype, panic_with_error, symbol_short, Address, Env};

use nester_common::ContractError;

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

#[contracttype]
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum Role {
    Admin,
    Operator,
}

#[contracttype]
#[derive(Clone)]
pub struct AdminTransfer {
    pub from: Address,
    pub to: Address,
}

// ---------------------------------------------------------------------------
// Internal storage keys
// ---------------------------------------------------------------------------

#[contracttype]
#[derive(Clone)]
enum DataKey {
    HasRole(Address, Role),
    PendingTransfer,
    AdminCount,
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

pub struct AccessControl;

impl AccessControl {
    /// Initialise access control. Must be called once per contract.
    pub fn initialize(env: &Env, admin: &Address) {
        if env.storage().instance().has(&DataKey::AdminCount) {
            panic_with_error!(env, ContractError::AlreadyInitialized);
        }
        admin.require_auth();
        internal_set_role(env, admin, Role::Admin, true);
        env.storage().instance().set(&DataKey::AdminCount, &1u32);
        env.events()
            .publish((symbol_short!("ac_init"), admin.clone()), Role::Admin);
    }

    /// Returns `true` if `account` holds `role`.
    pub fn has_role(env: &Env, account: &Address, role: Role) -> bool {
        env.storage()
            .instance()
            .get::<DataKey, bool>(&DataKey::HasRole(account.clone(), role))
            .unwrap_or(false)
    }

    /// Grant `role` to `grantee`. Caller must be Admin.
    pub fn grant_role(env: &Env, grantor: &Address, grantee: &Address, role: Role) {
        grantor.require_auth();
        Self::require_role(env, grantor, Role::Admin);
        let already_has = Self::has_role(env, grantee, role.clone());
        internal_set_role(env, grantee, role.clone(), true);
        if matches!(role, Role::Admin) && !already_has {
            internal_inc_admin_count(env);
        }
        env.events().publish(
            (symbol_short!("grant"), grantor.clone(), grantee.clone()),
            role,
        );
    }

    /// Revoke `role` from `target`. Caller must be Admin.
    /// Panics when revoking the last Admin.
    pub fn revoke_role(env: &Env, revoker: &Address, target: &Address, role: Role) {
        revoker.require_auth();
        Self::require_role(env, revoker, Role::Admin);
        if matches!(role, Role::Admin) {
            if internal_admin_count(env) <= 1 {
                panic_with_error!(env, ContractError::InvalidOperation);
            }
            internal_dec_admin_count(env);
        }
        internal_set_role(env, target, role.clone(), false);
        env.events().publish(
            (symbol_short!("revoke"), revoker.clone(), target.clone()),
            role,
        );
    }

    /// Panics with `Unauthorized` if `account` does not hold `role`.
    pub fn require_role(env: &Env, account: &Address, role: Role) {
        if !Self::has_role(env, account, role) {
            panic_with_error!(env, ContractError::Unauthorized);
        }
    }

    /// Panics if `account` holds neither Admin nor Operator.
    pub fn require_admin_or_operator(env: &Env, account: &Address) {
        if !Self::has_role(env, account, Role::Admin)
            && !Self::has_role(env, account, Role::Operator)
        {
            panic_with_error!(env, ContractError::Unauthorized);
        }
    }

    /// Step 1 of two-step admin transfer.
    pub fn transfer_admin(env: &Env, current_admin: &Address, new_admin: &Address) {
        current_admin.require_auth();
        Self::require_role(env, current_admin, Role::Admin);
        env.storage().instance().set(
            &DataKey::PendingTransfer,
            &AdminTransfer {
                from: current_admin.clone(),
                to: new_admin.clone(),
            },
        );
        env.events().publish(
            (
                symbol_short!("xfr_prop"),
                current_admin.clone(),
                new_admin.clone(),
            ),
            (),
        );
    }

    /// Step 2 of two-step admin transfer. Grants new Admin, revokes old.
    pub fn accept_admin(env: &Env, new_admin: &Address) {
        new_admin.require_auth();
        let proposal: AdminTransfer = env
            .storage()
            .instance()
            .get(&DataKey::PendingTransfer)
            .unwrap_or_else(|| panic_with_error!(env, ContractError::InvalidOperation));
        if proposal.to != *new_admin {
            panic_with_error!(env, ContractError::Unauthorized);
        }
        let already_admin = Self::has_role(env, new_admin, Role::Admin);
        internal_set_role(env, new_admin, Role::Admin, true);
        if !already_admin {
            internal_inc_admin_count(env);
        }
        internal_dec_admin_count(env);
        internal_set_role(env, &proposal.from, Role::Admin, false);
        env.storage().instance().remove(&DataKey::PendingTransfer);
        env.events().publish(
            (symbol_short!("xfr_acc"), new_admin.clone()),
            proposal.from.clone(),
        );
    }
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

fn internal_set_role(env: &Env, account: &Address, role: Role, active: bool) {
    env.storage()
        .instance()
        .set(&DataKey::HasRole(account.clone(), role), &active);
}

fn internal_admin_count(env: &Env) -> u32 {
    env.storage()
        .instance()
        .get(&DataKey::AdminCount)
        .unwrap_or(0u32)
}

fn internal_inc_admin_count(env: &Env) {
    let c = internal_admin_count(env);
    env.storage().instance().set(&DataKey::AdminCount, &(c + 1));
}

fn internal_dec_admin_count(env: &Env) {
    let c = internal_admin_count(env);
    env.storage().instance().set(&DataKey::AdminCount, &(c - 1));
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod test;
