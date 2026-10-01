package db

func schemaAuthorization() error {
	return execSchema("authorization", `
	CREATE TABLE IF NOT EXISTS admin_roles (
		key VARCHAR(100) PRIMARY KEY,
		description TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS admin_permissions (
		key VARCHAR(150) PRIMARY KEY,
		description TEXT NOT NULL DEFAULT ''
	);
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

	INSERT INTO admin_roles (key, description)
	VALUES ('super_admin', 'Full administrative access')
	ON CONFLICT (key) DO NOTHING;
	INSERT INTO admin_permissions (key, description)
	VALUES ('admin.access', 'Access administrative APIs')
	ON CONFLICT (key) DO NOTHING;
	INSERT INTO admin_role_permissions (role_key, permission_key)
	VALUES ('super_admin', 'admin.access')
	ON CONFLICT (role_key, permission_key) DO NOTHING;
	`)
}

// backfillAdminRoles runs after ensureAdmin so both existing legacy admins and
// the bootstrap admin receive an explicit role assignment.
func backfillAdminRoles() error {
	return execSchema("authorization_role_backfill", `
	INSERT INTO player_admin_roles (player_id, role_key)
	SELECT id, 'super_admin' FROM players WHERE COALESCE(is_admin, false)
	ON CONFLICT (player_id, role_key) DO NOTHING;
	`)
}
