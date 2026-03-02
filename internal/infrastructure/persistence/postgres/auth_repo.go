package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	authDomain "ai-auto-trade/internal/domain/auth"
	authinfra "ai-auto-trade/internal/infrastructure/auth"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AuthRepo 提供使用者、角色、權限的存取。
type AuthRepo struct {
	db *gorm.DB
}

// NewAuthRepo 建立 AuthRepo。
func NewAuthRepo(db *gorm.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

// FindByEmail 依 email 查詢使用者與主要角色。
func (r *AuthRepo) FindByEmail(ctx context.Context, email string) (authDomain.User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		return authDomain.User{}, err
	}

	// Fetch role
	var roleName string
	err = r.db.WithContext(ctx).Table("roles").
		Select("roles.name").
		Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", u.ID).
		Limit(1).
		Scan(&roleName).Error
	
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return authDomain.User{}, err
	}

	return authDomain.User{
		ID:       u.ID,
		Email:    u.Email,
		Name:     u.DisplayName,
		Password: u.PasswordHash,
		Status:   authDomain.Status(u.Status),
		Role:     authDomain.Role(roleName),
	}, nil
}

// FindByID 依 ID 查詢使用者與主要角色。
func (r *AuthRepo) FindByID(ctx context.Context, id string) (authDomain.User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		return authDomain.User{}, err
	}

	var roleName string
	err = r.db.WithContext(ctx).Table("roles").
		Select("roles.name").
		Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", u.ID).
		Limit(1).
		Scan(&roleName).Error
	
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return authDomain.User{}, err
	}

	return authDomain.User{
		ID:       u.ID,
		Email:    u.Email,
		Name:     u.DisplayName,
		Password: u.PasswordHash,
		Status:   authDomain.Status(u.Status),
		Role:     authDomain.Role(roleName),
	}, nil
}

// Create 建立新使用者並賦予預設角色。
func (r *AuthRepo) Create(ctx context.Context, u authDomain.User) (string, error) {
	var uid string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 建立使用者
		user := User{
			Email:        u.Email,
			DisplayName:  u.Name,
			PasswordHash: u.Password,
			Status:       "active",
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}},
			DoUpdates: clause.AssignmentColumns([]string{"display_name", "password_hash"}),
		}).Create(&user).Error; err != nil {
			return err
		}
		uid = user.ID

		// 2. 取得或建立使用者角色 ID
		var role Role
		if err := tx.Where("name = ?", string(authDomain.RoleUser)).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				role = Role{
					Name:         string(authDomain.RoleUser),
					Description:  "system role user",
					IsSystemRole: true,
				}
				if err := tx.Create(&role).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// 3. 綁定角色
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&UserRole{
			UserID: uid,
			RoleID: role.ID,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}
	return uid, nil
}

// SaveSession 寫入 refresh token session。
func (r *AuthRepo) SaveSession(ctx context.Context, sess authDomain.Session) error {
	m := AuthSession{
		UserID:         sess.UserID,
		RefreshTokenID: sess.Token,
		ExpiresAt:      sess.ExpiresAt,
		UserAgent:      sess.UserAgent,
		IPAddress:      sess.IPAddress,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "refresh_token_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"expires_at", "revoked_at", "user_agent", "ip_address"}),
	}).Create(&m).Error
}

// GetSession 依 refresh token 查詢 session。
func (r *AuthRepo) GetSession(ctx context.Context, token string) (authDomain.Session, error) {
	var m AuthSession
	err := r.db.WithContext(ctx).Where("refresh_token_id = ?", token).First(&m).Error
	if err != nil {
		return authDomain.Session{}, err
	}

	return authDomain.Session{
		UserID:    m.UserID,
		Token:     m.RefreshTokenID,
		ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt,
		UserAgent: m.UserAgent,
		IPAddress: m.IPAddress,
		CreatedAt: m.CreatedAt,
	}, nil
}

// RevokeSession 標記 refresh token 為失效。
func (r *AuthRepo) RevokeSession(ctx context.Context, token string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&AuthSession{}).
		Where("refresh_token_id = ?", token).
		Update("revoked_at", now).Error
}

// SeedDefaults 建立預設角色與帳號（admin/analyst/user）。
func (r *AuthRepo) SeedDefaults(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roleIDs := map[authDomain.Role]string{}
		roles := []authDomain.Role{authDomain.RoleAdmin, authDomain.RoleAnalyst, authDomain.RoleUser, authDomain.RoleService}
		
		for _, roleName := range roles {
			role := Role{
				Name:         string(roleName),
				Description:  fmt.Sprintf("system role %s", roleName),
				IsSystemRole: true,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"description"}),
			}).Create(&role).Error; err != nil {
				return err
			}
			roleIDs[roleName] = role.ID
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

			user := User{
				Email:        u.email,
				DisplayName:  u.name,
				PasswordHash: hash,
				Status:       "active",
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "email"}},
				DoUpdates: clause.AssignmentColumns([]string{"display_name", "password_hash"}),
			}).Create(&user).Error; err != nil {
				return err
			}

			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&UserRole{
				UserID: user.ID,
				RoleID: roleIDs[u.role],
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// IsPermissionGranted 確認使用者是否擁有權限。
func (r *AuthRepo) IsPermissionGranted(ctx context.Context, userID string, perm string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("user_roles").
		Select("1").
		Joins("JOIN role_permissions rp ON user_roles.role_id = rp.role_id").
		Joins("JOIN permissions p ON rp.permission_id = p.id").
		Where("user_roles.user_id = ? AND p.name = ?", userID, perm).
		Count(&count).Error
	
	return count > 0, err
}

// SeedPermissions 建立最小權限集合並賦予角色。
func (r *AuthRepo) SeedPermissions(ctx context.Context, perms []string, rolePerms map[authDomain.Role][]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		permIDs := map[string]string{}
		for _, pName := range perms {
			p := Permission{
				Name:        pName,
				Description: fmt.Sprintf("auto seeded perm %s", pName),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"description"}),
			}).Create(&p).Error; err != nil {
				return err
			}
			permIDs[pName] = p.ID
		}

		var roles []Role
		if err := tx.Find(&roles).Error; err != nil {
			return err
		}
		roleIDs := map[authDomain.Role]string{}
		for _, role := range roles {
			roleIDs[authDomain.Role(role.Name)] = role.ID
		}

		for role, plist := range rolePerms {
			rid, ok := roleIDs[role]
			if !ok {
				return fmt.Errorf("role %s not found when seeding permissions", role)
			}
			for _, pName := range plist {
				pid := permIDs[pName]
				if pid == "" {
					return fmt.Errorf("permission %s id missing", pName)
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&RolePermission{
					RoleID:       rid,
					PermissionID: pid,
				}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}
