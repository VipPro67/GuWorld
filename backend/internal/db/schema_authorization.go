package db

import "github.com/vippro67/GuWorld/backend/internal/authz"

func schemaAuthorization() error {
	if err := execSchema("authorization", `
	CREATE TABLE IF NOT EXISTS admin_roles (
		key VARCHAR(100) PRIMARY KEY,
		description TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS admin_permissions (
		key VARCHAR(150) PRIMARY KEY,
		description TEXT NOT NULL DEFAULT ''
	);
	-- Forward-patch: a briefly-wired UUID-based revision declared
	-- admin_permissions with a NOT NULL name column. Keep the seed valid
	-- on both shapes; nothing in live code reads this table's name.
	ALTER TABLE admin_permissions ADD COLUMN IF NOT EXISTS name VARCHAR(100) NOT NULL DEFAULT '';
	CREATE TABLE IF NOT EXISTS admin_role_permissions (
		role_key VARCHAR(100) NOT NULL REFERENCES admin_roles(key) ON DELETE CASCADE,
		permission_key VARCHAR(150) NOT NULL REFERENCES admin_permissions(key) ON DELETE CASCADE,
		PRIMARY KEY (role_key, permission_key)
	);
	CREATE TABLE IF NOT EXISTS player_admin_roles (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		role_key VARCHAR(100) NOT NULL REFERENCES admin_roles(key) ON DELETE CASCADE,
		PRIMARY KEY (player_id, role_key)
	);
	CREATE INDEX IF NOT EXISTS idx_player_admin_roles_role_key ON player_admin_roles(role_key);

	-- name/level are owned by the admin_roles step; include them so this
	-- seed is valid regardless of which step created the table first.
	INSERT INTO admin_roles (key, name, description, level)
	VALUES ('super_admin', 'Super Admin', 'Full administrative access', 100)
	ON CONFLICT (key) DO NOTHING;
	INSERT INTO admin_permissions (key, name, description)
	VALUES
		('admin.dashboard.access', 'Dashboard access', 'Access the administration dashboard'),
		('players.inventory.grant', 'Grant inventory', 'Grant inventory resources to players'),
		('admin.*', 'All permissions', 'All administration permissions')
	ON CONFLICT (key) DO NOTHING;
	INSERT INTO admin_role_permissions (role_key, permission_key)
	VALUES
		('admin', 'admin.dashboard.access'),
		('super_admin', 'admin.dashboard.access'),
		('super_admin', 'admin.*')
	ON CONFLICT (role_key, permission_key) DO NOTHING;
	`); err != nil {
		return err
	}
	for _, role := range authz.BuiltinRoles() {
		if _, err := DB.Exec(`DELETE FROM admin_role_permissions WHERE role_key=$1`, role.Key); err != nil {
			return err
		}
		for _, permission := range role.Permissions {
			if _, err := DB.Exec(`INSERT INTO admin_permissions (key, name, description) VALUES ($1,$1,'Built-in permission') ON CONFLICT (key) DO NOTHING`, permission); err != nil {
				return err
			}
			if _, err := DB.Exec(`INSERT INTO admin_role_permissions (role_key, permission_key) VALUES ($1,$2) ON CONFLICT DO NOTHING`, role.Key, permission); err != nil {
				return err
			}
		}
	}
	return nil
}

// backfillAdminRoles runs after ensureAdmin so both existing legacy admins and
// the bootstrap admin receive an explicit role assignment.
func backfillAdminRoles() error {
	return execSchema("authorization_role_backfill", `
	INSERT INTO player_admin_roles (player_id, role_key)
	SELECT id, 'admin' FROM players p WHERE COALESCE(is_admin, false)
	  AND NOT EXISTS (SELECT 1 FROM player_admin_roles par WHERE par.player_id=p.id)
	ON CONFLICT (player_id, role_key) DO NOTHING;
	`)
}
