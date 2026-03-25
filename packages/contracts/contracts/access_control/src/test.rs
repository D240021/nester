extern crate std;

use soroban_sdk::{contract, contractimpl, testutils::Address as _, Address, Env};

use crate::{AccessControl, Role};

#[contract]
struct TestAC;
#[contractimpl]
impl TestAC {}

fn setup() -> (Env, Address, Address, Address) {
    let env = Env::default();
    env.mock_all_auths();
    let admin = Address::generate(&env);
    let other = Address::generate(&env);
    let cid = env.register_contract(None, TestAC);
    env.as_contract(&cid, || AccessControl::initialize(&env, &admin));
    (env, admin, other, cid)
}

fn invoke(env: &Env, cid: &Address, f: impl FnOnce()) {
    env.as_contract(cid, f);
}
fn read<R>(env: &Env, cid: &Address, f: impl FnOnce() -> R) -> R {
    env.as_contract(cid, f)
}

#[test]
fn initialize_grants_admin() {
    let (env, admin, _, cid) = setup();
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &admin, Role::Admin)));
}

#[test]
fn initialize_does_not_grant_operator() {
    let (env, admin, _, cid) = setup();
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &admin, Role::Operator)));
}

#[test]
#[should_panic]
fn initialize_twice_panics() {
    let (env, admin, _, cid) = setup();
    invoke(&env, &cid, || AccessControl::initialize(&env, &admin));
}

#[test]
fn has_role_false_for_unknown() {
    let env = Env::default();
    env.mock_all_auths();
    let cid = env.register_contract(None, TestAC);
    let s = Address::generate(&env);
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &s, Role::Admin)));
}

#[test]
fn admin_can_grant_operator() {
    let (env, admin, op, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &op, Role::Operator)));
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &op, Role::Admin)));
}

#[test]
fn admin_can_grant_admin_to_another() {
    let (env, admin, second, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &second, Role::Admin));
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &second, Role::Admin)));
}

#[test]
#[should_panic]
fn non_admin_cannot_grant() {
    let (env, admin, op, cid) = setup();
    let outsider = Address::generate(&env);
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    invoke(&env, &cid, || AccessControl::grant_role(&env, &op, &outsider, Role::Operator));
}

#[test]
fn regrant_is_idempotent() {
    let (env, admin, op, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &op, Role::Operator)));
}

#[test]
fn admin_can_revoke_operator() {
    let (env, admin, op, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    invoke(&env, &cid, || AccessControl::revoke_role(&env, &admin, &op, Role::Operator));
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &op, Role::Operator)));
}

#[test]
fn admin_can_revoke_second_admin() {
    let (env, admin, second, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &second, Role::Admin));
    invoke(&env, &cid, || AccessControl::revoke_role(&env, &admin, &second, Role::Admin));
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &second, Role::Admin)));
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &admin, Role::Admin)));
}

#[test]
#[should_panic]
fn revoke_last_admin_panics() {
    let (env, admin, _, cid) = setup();
    invoke(&env, &cid, || AccessControl::revoke_role(&env, &admin, &admin, Role::Admin));
}

#[test]
#[should_panic]
fn non_admin_cannot_revoke() {
    let (env, admin, op, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    invoke(&env, &cid, || AccessControl::revoke_role(&env, &op, &admin, Role::Admin));
}

#[test]
fn require_role_passes() {
    let (env, admin, op, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    read(&env, &cid, || {
        AccessControl::require_role(&env, &admin, Role::Admin);
        AccessControl::require_role(&env, &op, Role::Operator);
    });
}

#[test]
#[should_panic]
fn require_role_panics_for_lacking_account() {
    let (env, _, other, cid) = setup();
    read(&env, &cid, || AccessControl::require_role(&env, &other, Role::Admin));
}

#[test]
fn two_step_transfer_happy_path() {
    let (env, admin, new_admin, cid) = setup();
    invoke(&env, &cid, || AccessControl::transfer_admin(&env, &admin, &new_admin));
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &admin, Role::Admin)));
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &new_admin, Role::Admin)));
    invoke(&env, &cid, || AccessControl::accept_admin(&env, &new_admin));
    assert!(read(&env, &cid, || AccessControl::has_role(&env, &new_admin, Role::Admin)));
    assert!(!read(&env, &cid, || AccessControl::has_role(&env, &admin, Role::Admin)));
}

#[test]
#[should_panic]
fn wrong_acceptor_panics() {
    let (env, admin, new_admin, cid) = setup();
    let imposter = Address::generate(&env);
    invoke(&env, &cid, || AccessControl::transfer_admin(&env, &admin, &new_admin));
    invoke(&env, &cid, || AccessControl::accept_admin(&env, &imposter));
}

#[test]
#[should_panic]
fn accept_without_proposal_panics() {
    let (env, _, other, cid) = setup();
    invoke(&env, &cid, || AccessControl::accept_admin(&env, &other));
}

#[test]
#[should_panic]
fn operator_cannot_grant() {
    let (env, admin, op, cid) = setup();
    let target = Address::generate(&env);
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    invoke(&env, &cid, || AccessControl::grant_role(&env, &op, &target, Role::Operator));
}

#[test]
#[should_panic]
fn operator_cannot_revoke() {
    let (env, admin, op, cid) = setup();
    invoke(&env, &cid, || AccessControl::grant_role(&env, &admin, &op, Role::Operator));
    invoke(&env, &cid, || AccessControl::revoke_role(&env, &op, &admin, Role::Admin));
}
