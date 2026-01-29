package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"ai-auto-trade/internal/application/trading"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

// TradingRepo 提供記憶體版儲存，僅供測試或無 DB 時使用。
type TradingRepo struct {
	mu         sync.Mutex
	strategies map[string]tradingDomain.Strategy
	backtests  map[string][]tradingDomain.BacktestRecord
	trades     []tradingDomain.TradeRecord
	positions  map[string]tradingDomain.Position
	logs       []tradingDomain.LogEntry
	reports    map[string][]tradingDomain.Report
}

// NewTradingRepo 建立記憶體實例。
func NewTradingRepo() *TradingRepo {
	return &TradingRepo{
		strategies: make(map[string]tradingDomain.Strategy),
		backtests:  make(map[string][]tradingDomain.BacktestRecord),
		positions:  make(map[string]tradingDomain.Position),
		reports:    make(map[string][]tradingDomain.Report),
	}
}

func (r *TradingRepo) nextID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func (r *TradingRepo) CreateStrategy(_ context.Context, s tradingDomain.Strategy) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID("st")
	now := time.Now()
	s.ID = id
	s.CreatedAt = now
	s.UpdatedAt = now
	r.strategies[id] = s
	return id, nil
}

func (r *TradingRepo) UpdateStrategy(_ context.Context, s tradingDomain.Strategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.strategies[s.ID]; !ok {
		return fmt.Errorf("strategy not found")
	}
	s.UpdatedAt = time.Now()
	r.strategies[s.ID] = s
	return nil
}

func (r *TradingRepo) GetStrategy(_ context.Context, id string) (tradingDomain.Strategy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.strategies[id]
	if !ok {
		return tradingDomain.Strategy{}, fmt.Errorf("strategy not found")
	}
	return s, nil
}

func (r *TradingRepo) ListStrategies(_ context.Context, filter trading.StrategyFilter) ([]tradingDomain.Strategy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]tradingDomain.Strategy, 0, len(r.strategies))
	for _, s := range r.strategies {
		if filter.Status != "" && s.Status != tradingDomain.Status(filter.Status) {
			continue
		}
		if filter.Env != "" && s.Env != tradingDomain.Environment(filter.Env) {
			continue
		}
		if filter.Name != "" && !containsFold(s.Name, filter.Name) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *TradingRepo) SetStatus(_ context.Context, id string, status tradingDomain.Status, env tradingDomain.Environment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.strategies[id]
	if !ok {
		return fmt.Errorf("strategy not found")
	}
	s.Status = status
	if env != "" {
		s.Env = env
	}
	s.UpdatedAt = time.Now()
	r.strategies[id] = s
	return nil
}

func (r *TradingRepo) DeleteStrategy(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.strategies[id]; !ok {
		return fmt.Errorf("strategy not found")
	}
	delete(r.strategies, id)
	delete(r.backtests, id)
	delete(r.reports, id)
	for key := range r.positions {
		if strings.HasPrefix(key, id+"|") {
			delete(r.positions, key)
		}
	}
	filterTrades := r.trades[:0]
	for _, t := range r.trades {
		if t.StrategyID != id {
			filterTrades = append(filterTrades, t)
		}
	}
	r.trades = filterTrades
	filterLogs := r.logs[:0]
	for _, l := range r.logs {
		if l.StrategyID != id {
			filterLogs = append(filterLogs, l)
		}
	}
	r.logs = filterLogs
	return nil
}

func (r *TradingRepo) SaveBacktest(_ context.Context, rec tradingDomain.BacktestRecord) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec.ID == "" {
		rec.ID = r.nextID("bt")
	}
	rec.CreatedAt = time.Now()
	r.backtests[rec.StrategyID] = append([]tradingDomain.BacktestRecord{rec}, r.backtests[rec.StrategyID]...)
	return rec.ID, nil
}

func (r *TradingRepo) ListBacktests(_ context.Context, strategyID string) ([]tradingDomain.BacktestRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.backtests[strategyID]
	out := make([]tradingDomain.BacktestRecord, len(list))
	copy(out, list)
	return out, nil
}

func (r *TradingRepo) SaveTrade(_ context.Context, trade tradingDomain.TradeRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if trade.ID == "" {
		trade.ID = r.nextID("tr")
	}
	if trade.CreatedAt.IsZero() {
		trade.CreatedAt = time.Now()
	}
	r.trades = append(r.trades, trade)
	return nil
}

func (r *TradingRepo) ListTrades(_ context.Context, filter tradingDomain.TradeFilter) ([]tradingDomain.TradeRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]tradingDomain.TradeRecord, 0, len(r.trades))
	for _, t := range r.trades {
		if filter.StrategyID != "" && t.StrategyID != filter.StrategyID {
			continue
		}
		if filter.Env != "" && t.Env != filter.Env {
			continue
		}
		if filter.StartDate != nil && t.EntryDate.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && t.EntryDate.After(*filter.EndDate) {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TradingRepo) GetOpenPosition(_ context.Context, strategyID string, env tradingDomain.Environment) (*tradingDomain.Position, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s|%s", strategyID, env)
	p, ok := r.positions[key]
	if !ok || p.Status != "open" {
		return nil, nil
	}
	return &p, nil
}

func (r *TradingRepo) GetPosition(_ context.Context, id string) (*tradingDomain.Position, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.positions {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("position not found")
}

func (r *TradingRepo) ListOpenPositions(_ context.Context) ([]tradingDomain.Position, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]tradingDomain.Position, 0, len(r.positions))
	for _, p := range r.positions {
		if p.Status == "open" {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *TradingRepo) UpsertPosition(_ context.Context, p tradingDomain.Position) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s|%s", p.StrategyID, p.Env)
	if p.ID == "" {
		p.ID = r.nextID("pos")
	}
	p.UpdatedAt = time.Now()
	r.positions[key] = p
	return nil
}

func (r *TradingRepo) ClosePosition(_ context.Context, id string, _ time.Time, _ float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, p := range r.positions {
		if p.ID == id {
			p.Status = "closed"
			p.UpdatedAt = time.Now()
			r.positions[key] = p
			return nil
		}
	}
	return fmt.Errorf("position not found")
}

func (r *TradingRepo) SaveLog(_ context.Context, log tradingDomain.LogEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if log.ID == "" {
		log.ID = r.nextID("log")
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	r.logs = append(r.logs, log)
	return nil
}

func (r *TradingRepo) ListLogs(_ context.Context, filter tradingDomain.LogFilter) ([]tradingDomain.LogEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	out := make([]tradingDomain.LogEntry, 0, len(r.logs))
	for i := len(r.logs) - 1; i >= 0; i-- { // 逆序，最近的先
		log := r.logs[i]
		if filter.StrategyID != "" && log.StrategyID != filter.StrategyID {
			continue
		}
		if filter.Env != "" && log.Env != filter.Env {
			continue
		}
		out = append(out, log)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *TradingRepo) SaveReport(_ context.Context, rep tradingDomain.Report) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rep.ID == "" {
		rep.ID = r.nextID("rep")
	}
	rep.CreatedAt = time.Now()
	r.reports[rep.StrategyID] = append([]tradingDomain.Report{rep}, r.reports[rep.StrategyID]...)
	return rep.ID, nil
}

func (r *TradingRepo) ListReports(_ context.Context, strategyID string) ([]tradingDomain.Report, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.reports[strategyID]
	out := make([]tradingDomain.Report, len(list))
	copy(out, list)
	return out, nil
}

func containsFold(haystack, needle string) bool {
	haystack = strings.ToLower(haystack)
	needle = strings.ToLower(needle)
	return strings.Contains(haystack, needle)
}

func (r *TradingRepo) LoadScoringStrategy(_ context.Context, _ string) (*strategyDomain.ScoringStrategy, error) {
	return nil, fmt.Errorf("LoadScoringStrategy not implemented in memory repo")
}

func (r *TradingRepo) ListActiveScoringStrategies(ctx context.Context) ([]*strategyDomain.ScoringStrategy, error) {
	return nil, nil
}
