package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-auto-trade/internal/domain/auth"
)

// UserRepository 存取使用者。
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (auth.User, error)
	FindByID(ctx context.Context, id string) (auth.User, error)
}

// PasswordHasher 驗證密碼。
type PasswordHasher interface {
	Compare(hashed, plain string) bool
}

// TokenIssuer 簽發/驗證 token。
type TokenIssuer interface {
	Issue(ctx context.Context, user auth.User) (auth.TokenPair, error)
	RevokeRefresh(ctx context.Context, token string) error
}

// Permission 表示功能權限。
type Permission string

const (
	PermUserManage            Permission = "user:manage"
	PermSystemHealth          Permission = "system:health"
	PermScreener              Permission = "screener:use"
	PermStrategy              Permission = "strategy:write"
	PermSubscription          Permission = "subscription:write"
	PermInternalAPI           Permission = "internal:ops"
	PermReportsFull           Permission = "reports:full"
	PermAnalysisQuery         Permission = "analysis_results.query"
	PermScreenerUse           Permission = "screener.use"
	PermIngestionTriggerDaily Permission = "ingestion.trigger_daily"
	PermAnalysisTriggerDaily  Permission = "analysis.trigger_daily"
)

// RolePermissions v1 簡化權限表。
var RolePermissions = map[auth.Role][]Permission{
	auth.RoleAdmin: {
		PermUserManage,
		PermSystemHealth,
		PermScreener,
		PermScreenerUse,
		PermAnalysisQuery,
		PermStrategy,
		PermSubscription,
		PermInternalAPI,
		PermReportsFull,
		PermIngestionTriggerDaily,
		PermAnalysisTriggerDaily,
	},
	auth.RoleAnalyst: {
		PermSystemHealth,
		PermScreener,
		PermScreenerUse,
		PermAnalysisQuery,
		PermStrategy,
		PermSubscription,
		PermReportsFull,
		PermIngestionTriggerDaily,
		PermAnalysisTriggerDaily,
	},
	auth.RoleUser: {
		PermScreener,
		PermScreenerUse,
		PermAnalysisQuery,
		PermStrategy,
		PermSubscription,
	},
	auth.RoleService: {
		PermInternalAPI,
		PermSystemHealth,
	},
}

// ResourceOwnerChecker 用於判斷資源是否屬於當前使用者。
type ResourceOwnerChecker interface {
	IsOwner(ctx context.Context, userID, resourceID string) bool
}

// AuthorizeInput 定義授權需求。
type AuthorizeInput struct {
	UserID     string
	Required   []Permission
	ResourceID string // 若需要判斷 owner
	OwnerPerm  Permission
}

// AuthorizeResult 回傳授權結果。
type AuthorizeResult struct {
	Allowed bool
	Reason  string
}

// LoginUseCase 驗證帳密並簽發 token。
type LoginUseCase struct {
	users  UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
	now    func() time.Time
}

func NewLoginUseCase(users UserRepository, hasher PasswordHasher, tokens TokenIssuer) *LoginUseCase {
	return &LoginUseCase{
		users:  users,
		hasher: hasher,
		tokens: tokens,
		now:    time.Now,
	}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	User  auth.User
	Token auth.TokenPair
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginResult, error) {
	var out LoginResult
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if email == "" || input.Password == "" {
		return out, errors.New("email and password required")
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return out, fmt.Errorf("find user: %w", err)
	}
	if !user.IsActive() {
		return out, errors.New("user disabled or locked")
	}
	if !uc.hasher.Compare(user.Password, input.Password) {
		return out, errors.New("invalid credentials")
	}

	token, err := uc.tokens.Issue(ctx, user)
	if err != nil {
		return out, fmt.Errorf("issue token: %w", err)
	}

	out.User = user
	out.Token = token
	return out, nil
}

// LogoutUseCase 處理 refresh token 作廢。
type LogoutUseCase struct {
	tokens TokenIssuer
}

func NewLogoutUseCase(tokens TokenIssuer) *LogoutUseCase {
	return &LogoutUseCase{tokens: tokens}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return errors.New("refresh token required")
	}
	return uc.tokens.RevokeRefresh(ctx, refreshToken)
}

// Authorizer 檢查角色/權限。
type Authorizer struct {
	users UserRepository
	owner ResourceOwnerChecker
}

func NewAuthorizer(users UserRepository, owner ResourceOwnerChecker) *Authorizer {
	return &Authorizer{users: users, owner: owner}
}

func (a *Authorizer) HasPermission(role auth.Role, perm Permission) bool {
	perms := RolePermissions[role]
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// Authorize 檢查使用者是否具備所需權限，並視情況檢查 owner。
func (a *Authorizer) Authorize(ctx context.Context, input AuthorizeInput) (AuthorizeResult, error) {
	user, err := a.users.FindByID(ctx, input.UserID)
	if err != nil {
		return AuthorizeResult{Allowed: false, Reason: "user not found"}, err
	}
	if !user.IsActive() {
		return AuthorizeResult{Allowed: false, Reason: "user disabled"}, nil
	}

	for _, perm := range input.Required {
		if a.HasPermission(user.Role, perm) {
			continue
		}
		// 若指定 owner 權限檢查且資源為本人
		if input.OwnerPerm != "" && input.ResourceID != "" && a.owner != nil {
			if a.owner.IsOwner(ctx, user.ID, input.ResourceID) && perm == input.OwnerPerm {
				continue
			}
		}
		return AuthorizeResult{Allowed: false, Reason: fmt.Sprintf("missing permission %s", perm)}, nil
	}

	return AuthorizeResult{Allowed: true}, nil
}
