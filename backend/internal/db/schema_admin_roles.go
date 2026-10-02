package db

import "github.com/vippro67/GuWorld/backend/internal/authz"

// Administrative roles are the authority for granting and revoking admin
// access. players.is_admin remains a compatibility projection for legacy
// clients and middleware.
func schemaAdminRoles() error {
	if err := execSchema("admin_roles", `
	CREATE TABLE IF NOT EXISTS admin_roles (
		key VARCHAR(50) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		level INTEGER NOT NULL UNIQUE CHECK (level > 0)
	);`); err != nil {
		return err
	}

	// Forward-patch databases created before name/level existed (e.g. by the
	// legacy authorization step which only declared key/description).
	if err := execSchema("admin_roles_patch_columns", `
	ALTER TABLE admin_roles ADD COLUMN IF NOT EXISTS name VARCHAR(100) NOT NULL DEFAULT '';
	ALTER TABLE admin_roles ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
	ALTER TABLE admin_roles ADD COLUMN IF NOT EXISTS level INTEGER DEFAULT 50;
	UPDATE admin_roles SET name = key WHERE name = '';
	UPDATE admin_roles SET level = 100 WHERE key = 'super_admin' AND (level IS NULL OR level <> 100);
	UPDATE admin_roles SET level = 50 WHERE key = 'admin' AND (level IS NULL OR level <> 50);`); err != nil {
		return err
	}

	if err := execSchema("player_admin_roles", `
	CREATE TABLE IF NOT EXISTS player_admin_roles (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		role_key VARCHAR(50) NOT NULL REFERENCES admin_roles(key) ON DELETE RESTRICT,
		assigned_by UUID REFERENCES players(id) ON DELETE SET NULL,
		assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, role_key)
	);
	CREATE INDEX IF NOT EXISTS idx_player_admin_roles_role_key
		ON player_admin_roles (role_key, player_id);`); err != nil {
		return err
	}

	if err := execSchema("admin_roles_seed", `
	INSERT INTO admin_roles (key, name, description, level) VALUES
		('admin', 'Admin', 'Quản trị viên', 50),
		('super_admin', 'Super Admin', 'Quản trị viên cấp cao nhất', 100)
	ON CONFLICT (key) DO UPDATE SET
		name = EXCLUDED.name,
		description = EXCLUDED.description,
		level = EXCLUDED.level;`); err != nil {
		return err
	}
	for _, role := range authz.BuiltinRoles() {
		if _, err := DB.Exec(`INSERT INTO admin_roles (key, name, description, level) VALUES ($1,$2,$3,$4)
			ON CONFLICT (key) DO UPDATE SET name=EXCLUDED.name, description=EXCLUDED.description, level=EXCLUDED.level`, role.Key, role.Name, role.Description, role.Level); err != nil {
			return err
		}
	}

	// Preserve access for legacy admins without accidentally making every
	// system account a super admin.
	return execSchema("admin_roles_legacy_backfill", `
	INSERT INTO player_admin_roles (player_id, role_key)
	SELECT p.id, 'admin' FROM players p
	WHERE COALESCE(p.is_admin, false)
	  AND NOT EXISTS (SELECT 1 FROM player_admin_roles par WHERE par.player_id = p.id)
	ON CONFLICT (player_id, role_key) DO NOTHING;`)
}

func ensureBootstrapAdminRole() error {
	if _, err := DB.Exec(`
	INSERT INTO player_admin_roles (player_id, role_key)
	SELECT id, 'super_admin' FROM players
	WHERE username = 'admin'
	  AND NOT EXISTS (SELECT 1 FROM player_admin_roles WHERE role_key = 'super_admin')
	ON CONFLICT (player_id, role_key) DO NOTHING;`); err != nil {
		return err
	}
	// Sync only divergent rows. The unguarded full-table rewrite dirtied every
	// player row on each boot (WAL storm + row locks), tripping the 15s
	// statement_timeout once players grows and blocking concurrent logins.
	_, err := DB.Exec(`
	UPDATE players p SET is_admin = EXISTS (
		SELECT 1 FROM player_admin_roles par WHERE par.player_id = p.id
	) WHERE p.is_admin IS DISTINCT FROM (EXISTS (
		SELECT 1 FROM player_admin_roles par WHERE par.player_id = p.id
	))`)
	return err
}
