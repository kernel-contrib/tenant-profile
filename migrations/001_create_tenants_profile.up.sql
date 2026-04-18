CREATE TABLE tenants_profile (
    id UUID,
    address JSONB NOT NULL DEFAULT '{}',
    industry TEXT,
    size TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_tenants_profile_id ON tenants_profile(id);
CREATE INDEX idx_tenants_profile_industry ON tenants_profile(industry)
WHERE deleted_at IS NULL;
