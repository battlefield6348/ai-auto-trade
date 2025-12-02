package auth

import "errors"

// Role 定義系統角色。
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleAnalyst Role = "analyst"
	RoleUser    Role = "user"
	RoleService Role = "service"
)

// Status 定義帳號狀態。
type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusLocked   Status = "locked"
)

// User 基本帳號資料。
type User struct {
	ID       string
	Email    string
	Name     string
	Role     Role
	Status   Status
	Password string // 雜湊後密碼
}

// Validate 基本欄位檢查。
func (u User) Validate() error {
	if u.ID == "" {
		return errors.New("id is required")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.Role == "" {
		return errors.New("role is required")
	}
	if u.Status == "" {
		return errors.New("status is required")
	}
	return nil
}

// IsActive 檢查是否可登入。
func (u User) IsActive() bool {
	return u.Status == StatusActive
}
