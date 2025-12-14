package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	authDomain "ai-auto-trade/internal/domain/auth"
	authinfra "ai-auto-trade/internal/infrastructure/auth"
)

// AuthRepo 提供使用者、角色、權限的存取。
type AuthRepo struct {
	db *sql.DB
}

// NewAuthRepo 建立 AuthRepo。
func NewAuthRepo(db *sql.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

// FindByEmail 依 email 查詢使用者與主要角色。
func (r *AuthRepo) FindByEmail(ctx context.Context, email string) (authDomain.User, error) {
	const q = `
SELECT u.id, u.email, u.display_name, u.password_hash, u.status, COALESCE(r.name, '') AS role_name
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.email = $1
LIMIT 1;
`
	var u authDomain.User
	var roleName string
	if err := r.db.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.Status, &roleName); err != nil {
		return authDomain.User{}, err
	}
	u.Role = authDomain.Role(roleName)
	return u, nil
}

// FindByID 依 ID 查詢使用者與主要角色。
func (r *AuthRepo) FindByID(ctx context.Context, id string) (authDomain.User, error) {
	const q = `
SELECT u.id, u.email, u.display_name, u.password_hash, u.status, COALESCE(r.name, '') AS role_name
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.id = $1
LIMIT 1;
`
	var u authDomain.User
	var roleName string
	if err := r.db.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.Status, &roleName); err != nil {
		return authDomain.User{}, err
	}
	u.Role = authDomain.Role(roleName)
	return u, nil
}

// SeedDefaults 建立預設角色與帳號（admin/analyst/user）。
func (r *AuthRepo) SeedDefaults(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	roleIDs := map[authDomain.Role]string{}
	roles := []authDomain.Role{authDomain.RoleAdmin, authDomain.RoleAnalyst, authDomain.RoleUser, authDomain.RoleService}
	for _, role := range roles {
		id, err := upsertRoleTx(ctx, tx, string(role))
		if err != nil {
			return err
		}
		roleIDs[role] = id
	}

	users := []struct {
		email string
		name  string
		role  authDomain.Role
	}{
		{"admin@example.com", "Admin", authDomain.RoleAdmin},
		{"analyst@example.com", "Analyst", authDomain.RoleAnalyst},
		{"user@example.com", "User", authDomain.RoleUser},
	}
	for _, u := range users {
		hash, err := authinfra.HashPassword("password123")
		if err != nil {
			return err
		}
		uid, err := upsertUserTx(ctx, tx, u.email, u.name, hash)
		if err != nil {
			return err
		}
		if err := attachRoleTx(ctx, tx, uid, roleIDs[u.role]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func upsertRoleTx(ctx context.Context, tx *sql.Tx, name string) (string, error) {
	const q = `
INSERT INTO roles (name, description, is_system_role)
VALUES ($1, $2, TRUE)
ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description
RETURNING id;
`
	var id string
	if err := tx.QueryRowContext(ctx, q, name, fmt.Sprintf("system role %s", name)).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func upsertUserTx(ctx context.Context, tx *sql.Tx, email, name, passwordHash string) (string, error) {
	const q = `
INSERT INTO users (email, display_name, password_hash, status)
VALUES ($1, $2, $3, 'active')
ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name, password_hash = EXCLUDED.password_hash
RETURNING id;
`
	var id string
	if err := tx.QueryRowContext(ctx, q, email, name, passwordHash).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func attachRoleTx(ctx context.Context, tx *sql.Tx, userID, roleID string) error {
	const q = `
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT (user_id, role_id) DO NOTHING;
`
	_, err := tx.ExecContext(ctx, q, userID, roleID)
	return err
}

// IsPermissionGranted 確認使用者是否擁有權限。
func (r *AuthRepo) IsPermissionGranted(ctx context.Context, userID string, perm string) (bool, error) {
	const q = `
SELECT EXISTS (
	SELECT 1
	FROM user_roles ur
	JOIN role_permissions rp ON ur.role_id = rp.role_id
	JOIN permissions p ON rp.permission_id = p.id
	WHERE ur.user_id = $1 AND p.name = $2
);
`
	var ok bool
	if err := r.db.QueryRowContext(ctx, q, userID, perm).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

// SeedPermissions 建立最小權限集合並賦予角色。
func (r *AuthRepo) SeedPermissions(ctx context.Context, perms []string, rolePerms map[authDomain.Role][]string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	permIDs := map[string]string{}
	for _, p := range perms {
		id, err := upsertPermissionTx(ctx, tx, p)
		if err != nil {
			return err
		}
		permIDs[p] = id
	}

	roleIDs := map[authDomain.Role]string{}
	rows, err := tx.QueryContext(ctx, `SELECT id, name FROM roles`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		roleIDs[authDomain.Role(name)] = id
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for role, plist := range rolePerms {
		rid, ok := roleIDs[role]
		if !ok {
			return fmt.Errorf("role %s not found when seeding permissions", role)
		}
		for _, p := range plist {
			pid := permIDs[p]
			if pid == "" {
				return fmt.Errorf("permission %s id missing", p)
			}
			if err := attachRolePermissionTx(ctx, tx, rid, pid); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
	}

	return tx.Commit()
}

func upsertPermissionTx(ctx context.Context, tx *sql.Tx, name string) (string, error) {
	const q = `
INSERT INTO permissions (name, description)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description
RETURNING id;
`
	var id string
	if err := tx.QueryRowContext(ctx, q, name, fmt.Sprintf("auto seeded perm %s", name)).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func attachRolePermissionTx(ctx context.Context, tx *sql.Tx, roleID, permID string) error {
	const q = `
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT (role_id, permission_id) DO NOTHING;
`
	_, err := tx.ExecContext(ctx, q, roleID, permID)
	return err
}
