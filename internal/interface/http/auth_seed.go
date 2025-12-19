package httpapi

import (
	"context"
	"fmt"

	"ai-auto-trade/internal/application/auth"
	authDomain "ai-auto-trade/internal/domain/auth"
)

// seedAuth 將預設角色、權限與帳號寫入儲存層（若支援）。
func seedAuth(ctx context.Context, repo auth.UserRepository) error {
	ar, ok := repo.(interface {
		SeedDefaults(ctx context.Context) error
		SeedPermissions(ctx context.Context, perms []string, rolePerms map[authDomain.Role][]string) error
	})
	if !ok {
		return fmt.Errorf("auth repository does not support seeding")
	}

	// 建立基本帳號與角色
	if err := ar.SeedDefaults(ctx); err != nil {
		return fmt.Errorf("seed defaults: %w", err)
	}

	// 建立權限與映射
	allPerms := []string{
		string(auth.PermUserManage),
		string(auth.PermSystemHealth),
		string(auth.PermScreener),
		string(auth.PermScreenerUse),
		string(auth.PermAnalysisQuery),
		string(auth.PermStrategy),
		string(auth.PermSubscription),
		string(auth.PermInternalAPI),
		string(auth.PermReportsFull),
		string(auth.PermIngestionTriggerDaily),
		string(auth.PermIngestionTriggerBackfill),
		string(auth.PermAnalysisTriggerDaily),
	}
	if err := ar.SeedPermissions(ctx, allPerms, auth.RolePermissionsAsStrings()); err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}

	return nil
}
