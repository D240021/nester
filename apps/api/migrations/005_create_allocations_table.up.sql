CREATE TABLE IF NOT EXISTS allocations (
    id UUID PRIMARY KEY,
    vault_id UUID NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    protocol TEXT NOT NULL CHECK (char_length(protocol) > 0),
    amount NUMERIC(20,8) NOT NULL CHECK (amount >= 0),
    apy NUMERIC(10,4) NOT NULL CHECK (apy >= 0),
    allocated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_allocations_vault_id ON allocations (vault_id);
