CREATE TABLE tenants_profile (
    tenant_id UUID PRIMARY KEY,
    address JSONB NOT NULL DEFAULT '{}',
    industry TEXT,
    size TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_tenants_profile_industry ON tenants_profile(industry)
WHERE deleted_at IS NULL;
