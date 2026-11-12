package db

func schemaEffectCompositions() error {
	return execSchema("admin_effect_composition_drafts", `
	CREATE TABLE IF NOT EXISTS admin_effect_composition_drafts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(120) NOT NULL,
		product_profile VARCHAR(80) NOT NULL,
		status VARCHAR(16) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ARCHIVED')),
		definition JSONB NOT NULL,
		created_by UUID REFERENCES players(id) ON DELETE SET NULL,
		updated_by UUID REFERENCES players(id) ON DELETE SET NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_admin_effect_composition_drafts_profile
		ON admin_effect_composition_drafts (product_profile, updated_at DESC)
		WHERE status = 'DRAFT';`)
}
