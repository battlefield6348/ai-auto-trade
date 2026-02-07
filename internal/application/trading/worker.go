package trading

import (
	"context"
	"log"
	"time"

	tradingDomain "ai-auto-trade/internal/domain/trading"
)

// BackgroundWorker 定期執行自動交易。
type BackgroundWorker struct {
	svc      *Service
	interval time.Duration
	stopChan chan struct{}
}

// NewBackgroundWorker 建立背景工作者。
func NewBackgroundWorker(svc *Service, interval time.Duration) *BackgroundWorker {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	return &BackgroundWorker{
		svc:      svc,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start 啟動迴圈。
func (w *BackgroundWorker) Start() {
	log.Printf("[Worker] Starting trading background worker with interval: %v", w.interval)
	ticker := time.NewTicker(w.interval)
	go func() {
		// 啟動後立即執行一次
		w.runOnce()

		for {
			select {
			case <-ticker.C:
				w.runOnce()
			case <-w.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop 停止迴圈。
func (w *BackgroundWorker) Stop() {
	close(w.stopChan)
}

func (w *BackgroundWorker) runOnce() {
	ctx := context.Background()
	log.Printf("[Worker] Checking active strategies for auto-trade...")

	strats, err := w.svc.repo.ListActiveScoringStrategies(ctx)
	if err != nil {
		log.Printf("[Worker] Failed to list active strategies: %v", err)
		return
	}

	if len(strats) == 0 {
		log.Printf("[Worker] No active scoring strategies found.")
		return
	}

	for _, s := range strats {
		log.Printf("[Worker] Executing strategy: %s (Slug: %s, Env: %s)", s.Name, s.Slug, s.Env)
		
		envs := []tradingDomain.Environment{}
		switch s.Env {
		case "prod":
			envs = append(envs, tradingDomain.EnvProd)
		case "real":
			envs = append(envs, tradingDomain.EnvReal)
		case "paper":
			envs = append(envs, tradingDomain.EnvPaper)
		case "test":
			envs = append(envs, tradingDomain.EnvTest)
		case "both":
			envs = append(envs, tradingDomain.EnvPaper, tradingDomain.EnvTest)
		default:
			envs = append(envs, tradingDomain.EnvTest)
		}

		// userID 使用策略擁有者的 ID
		userID := s.UserID
		if userID == "" {
			userID = "00000000-0000-0000-0000-000000000001" // Fallback to admin
		}
		
		for _, env := range envs {
			err := w.svc.ExecuteScoringAutoTrade(ctx, s.Slug, env, userID)
			if err != nil {
				log.Printf("[Worker] Strategy %s (%s) execution failed: %v", s.Slug, env, err)
			} else {
				log.Printf("[Worker] Strategy %s (%s) execution completed.", s.Slug, env)
			}
		}
	}
}

