package db

import "fmt"

// Admin RBAC schema. The legacy players.is_admin flag remains in place during
// the transition and is backfilled into these tables by seedAdminRBAC.
func schemaAdminRBAC() error {
	if err := execSchema("admin_roles", `
	CREATE TABLE IF NOT EXISTS admin_roles (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		key VARCHAR(64) NOT NULL UNIQUE,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		is_system BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("admin_permissions", `
	CREATE TABLE IF NOT EXISTS admin_permissions (
		key VARCHAR(128) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		is_system BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("admin_role_permissions", `
	CREATE TABLE IF NOT EXISTS admin_role_permissions (
		role_id UUID NOT NULL REFERENCES admin_roles(id) ON DELETE CASCADE,
		permission_key VARCHAR(128) NOT NULL REFERENCES admin_permissions(key) ON DELETE CASCADE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (role_id, permission_key)
	);`); err != nil {
		return err
	}

	if err := execSchema("player_admin_roles", `
	CREATE TABLE IF NOT EXISTS player_admin_roles (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		role_id UUID NOT NULL REFERENCES admin_roles(id) ON DELETE CASCADE,
		assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, role_id)
	);`); err != nil {
		return err
	}

	indexes := []struct {
		name string
		sql  string
	}{
		{"player_admin_roles_player_id_idx", `CREATE INDEX IF NOT EXISTS idx_player_admin_roles_player_id ON player_admin_roles (player_id);`},
		{"player_admin_roles_role_id_idx", `CREATE INDEX IF NOT EXISTS idx_player_admin_roles_role_id ON player_admin_roles (role_id);`},
		{"admin_role_permissions_role_id_idx", `CREATE INDEX IF NOT EXISTS idx_admin_role_permissions_role_id ON admin_role_permissions (role_id);`},
		{"admin_role_permissions_permission_key_idx", `CREATE INDEX IF NOT EXISTS idx_admin_role_permissions_permission_key ON admin_role_permissions (permission_key);`},
	}
	for _, index := range indexes {
		if err := execSchema(index.name, index.sql); err != nil {
			return err
		}
	}

	return nil
}

// seedAdminRBAC installs the built-in unrestricted role and migrates every
// legacy admin to it. ON CONFLICT makes the whole operation safe on restarts.
func seedAdminRBAC() (err error) {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("begin admin RBAC seed: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	statements := []string{
		`INSERT INTO admin_roles (key, name, description, is_system)
		 VALUES ('super_admin', 'Super Admin', 'Unrestricted access to all administration features.', true)
		 ON CONFLICT (key) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_system = true,
			updated_at = NOW();`,
		`INSERT INTO admin_permissions (key, name, description, is_system)
		 VALUES ('admin.*', 'All admin permissions', 'Grants every administration permission.', true)
		 ON CONFLICT (key) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_system = true;`,
		`INSERT INTO admin_role_permissions (role_id, permission_key)
		 SELECT id, 'admin.*' FROM admin_roles WHERE key = 'super_admin'
		 ON CONFLICT (role_id, permission_key) DO NOTHING;`,
		`INSERT INTO player_admin_roles (player_id, role_id)
		 SELECT players.id, admin_roles.id
		 FROM players
		 CROSS JOIN admin_roles
		 WHERE COALESCE(players.is_admin, false) AND admin_roles.key = 'super_admin'
		 ON CONFLICT (player_id, role_id) DO NOTHING;`,
	}
	for _, statement := range statements {
		if _, err = tx.Exec(statement); err != nil {
			return fmt.Errorf("seed admin RBAC: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit admin RBAC seed: %w", err)
	}
	return nil
}
