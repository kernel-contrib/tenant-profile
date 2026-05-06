-- Tenant profiles: 1:1 extension data for tenants (address, industry, size).
CREATE TABLE tenant_profiles (
    tenant_id UUID PRIMARY KEY,
    address JSONB NOT NULL DEFAULT '{}',
    industry TEXT,
    size TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_tenant_profiles_industry ON tenant_profiles (industry)
WHERE deleted_at IS NULL;
-- Auto-update updated_at trigger.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_tenant_profiles_updated_at BEFORE
UPDATE ON tenant_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();
