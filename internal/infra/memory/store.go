package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	authDomain "ai-auto-trade/internal/domain/auth"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	authinfra "ai-auto-trade/internal/infrastructure/auth"
)

// Store 為 MVP 使用的記憶體資料庫，僅供示範且非併發安全。
type Store struct {
	mu              sync.RWMutex
	users           map[string]authDomain.User
	passwords       map[string]string
	tokens          map[string]tokenRecord
	sessions        map[string]sessionRecord
	tradingPairs    map[string]pairRecord                                    // id -> record
	pairByCode      map[string]string                                        // pair+market -> id
	dailyPrices     map[string]map[string]dataDomain.DailyPrice              // date -> stockID -> price
	analysisResults map[string]map[string]analysisDomain.DailyAnalysisResult // date -> stockID -> result
	backtestPreset  map[string][]byte
	idSeq           int64
}

type tokenRecord struct {
	UserID  string
	Expires time.Time
}

type sessionRecord struct {
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IPAddress string
	CreatedAt time.Time
}

type pairRecord struct {
	ID        string
	Pair      string
	Market    dataDomain.Market
	Name      string
	Industry  string
	CreatedAt time.Time
}

// NewStore 建立新的記憶體 Store 實例。
func NewStore() *Store {
	return &Store{
		users:           make(map[string]authDomain.User),
		passwords:       make(map[string]string),
		tokens:          make(map[string]tokenRecord),
		sessions:        make(map[string]sessionRecord),
		tradingPairs:    make(map[string]pairRecord),
		pairByCode:      make(map[string]string),
		dailyPrices:     make(map[string]map[string]dataDomain.DailyPrice),
		analysisResults: make(map[string]map[string]analysisDomain.DailyAnalysisResult),
		backtestPreset:  make(map[string][]byte),
	}
}

// ID generator (simple incremental).
func (s *Store) nextID() string {
	s.idSeq++
	return fmt.Sprintf("id-%d", s.idSeq)
}

// SeedUsers 建立預設帳號供登入測試。
func (s *Store) SeedUsers() {
	hash := func(p string) string {
		h, err := authinfra.HashPassword(p)
		if err != nil {
			return p
		}
		return h
	}
	s.addUser("admin@example.com", hash("password123"), "Admin", authDomain.RoleAdmin)
	s.addUser("analyst@example.com", hash("password123"), "Analyst", authDomain.RoleAnalyst)
	s.addUser("user@example.com", hash("password123"), "User", authDomain.RoleUser)
}

func (s *Store) addUser(email, password, name string, role authDomain.Role) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID()
	user := authDomain.User{
		ID:       id,
		Email:    email,
		Name:     name,
		Role:     role,
		Status:   authDomain.StatusActive,
		Password: password,
	}
	s.users[id] = user
	s.passwords[email] = password
}

// UserRepository impl
// FindByEmail 依 email 查詢使用者。
func (s *Store) FindByEmail(ctx context.Context, email string) (authDomain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return authDomain.User{}, fmt.Errorf("user not found")
}

// FindByID 依 ID 查詢使用者。
func (s *Store) FindByID(ctx context.Context, id string) (authDomain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return authDomain.User{}, fmt.Errorf("user not found")
	}
	return u, nil
}

// PasswordHasher impl (plain compare for MVP).
type PlainHasher struct{}

func (PlainHasher) Compare(hashed, plain string) bool {
	return hashed == plain
}

// TokenIssuer impl.
type MemoryTokenIssuer struct {
	store *Store
	ttl   time.Duration
}

// NewMemoryTokenIssuer 建立簡易的記憶體版 token 簽發器。
func NewMemoryTokenIssuer(store *Store, ttl time.Duration) *MemoryTokenIssuer {
	return &MemoryTokenIssuer{store: store, ttl: ttl}
}

func (m *MemoryTokenIssuer) Issue(ctx context.Context, user authDomain.User, _ authDomain.TokenMeta) (authDomain.TokenPair, error) {
	token := fmt.Sprintf("token-%s-%d", user.ID, time.Now().UnixNano())
	m.store.mu.Lock()
	m.store.tokens[token] = tokenRecord{
		UserID:  user.ID,
		Expires: time.Now().Add(m.ttl),
	}
	m.store.mu.Unlock()
	return authDomain.TokenPair{
		AccessToken:   token,
		RefreshToken:  token,
		AccessExpiry:  time.Now().Add(m.ttl),
		RefreshExpiry: time.Now().Add(m.ttl),
	}, nil
}

func (m *MemoryTokenIssuer) Refresh(ctx context.Context, token string) (authDomain.TokenPair, error) {
	rec, ok := m.store.sessions[token]
	if !ok || rec.ExpiresAt.Before(time.Now()) || (rec.RevokedAt != nil && !rec.RevokedAt.IsZero()) {
		return authDomain.TokenPair{}, fmt.Errorf("session not found")
	}
	user, ok := m.store.users[rec.UserID]
	if !ok {
		return authDomain.TokenPair{}, fmt.Errorf("user not found")
	}
	return m.Issue(ctx, user, authDomain.TokenMeta{})
}

func (m *MemoryTokenIssuer) RevokeRefresh(ctx context.Context, token string) error {
	return nil
}

// ValidateToken 驗證 access token 並回傳對應使用者。
func (s *Store) ValidateToken(token string) (authDomain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.tokens[token]
	if !ok || time.Now().After(rec.Expires) {
		return authDomain.User{}, false
	}
	u, ok := s.users[rec.UserID]
	return u, ok
}

// ResourceOwnerChecker impl: for MVP treat user-owned resources by userID match.
type OwnerChecker struct{}

func (OwnerChecker) IsOwner(ctx context.Context, userID, resourceID string) bool {
	return userID == resourceID
}

// BacktestPresetStore impl
func (s *Store) Save(ctx context.Context, preset []byte, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backtestPreset[userID] = preset
	return nil
}

func (s *Store) Load(ctx context.Context, userID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.backtestPreset[userID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return p, nil
}

// SessionStore impl
func (s *Store) SaveSession(ctx context.Context, sess authDomain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.Token] = sessionRecord{
		UserID:    sess.UserID,
		ExpiresAt: sess.ExpiresAt,
		RevokedAt: sess.RevokedAt,
		UserAgent: sess.UserAgent,
		IPAddress: sess.IPAddress,
		CreatedAt: sess.CreatedAt,
	}
	return nil
}

func (s *Store) GetSession(ctx context.Context, token string) (authDomain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.sessions[token]
	if !ok {
		return authDomain.Session{}, fmt.Errorf("session not found")
	}
	return authDomain.Session{
		Token:     token,
		UserID:    rec.UserID,
		ExpiresAt: rec.ExpiresAt,
		RevokedAt: rec.RevokedAt,
		UserAgent: rec.UserAgent,
		IPAddress: rec.IPAddress,
		CreatedAt: rec.CreatedAt,
	}, nil
}

func (s *Store) RevokeSession(ctx context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.sessions[token]
	if !ok {
		return fmt.Errorf("session not found")
	}
	now := time.Now()
	rec.RevokedAt = &now
	s.sessions[token] = rec
	return nil
}

// AnalysisQueryRepository impls
// FindByDate 依交易日期查詢分析結果，支援分頁與成功過濾。
func (s *Store) FindByDate(ctx context.Context, date time.Time, filter analysis.QueryFilter, sortOpt analysis.SortOption, pagination analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dateKey := date.Format("2006-01-02")
	results := s.analysisResults[dateKey]
	var list []analysisDomain.DailyAnalysisResult
	for _, r := range results {
		if filter.OnlySuccess && !r.Success {
			continue
		}
		list = append(list, r)
	}
	total := len(list)
	if pagination.Offset > total {
		return []analysisDomain.DailyAnalysisResult{}, total, nil
	}
	end := pagination.Offset + pagination.Limit
	if end > total {
		end = total
	}
	return list[pagination.Offset:end], total, nil
}

// FindHistory 依股票代碼與日期區間查詢歷史分析結果。
func (s *Store) FindHistory(ctx context.Context, symbol string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var all []analysisDomain.DailyAnalysisResult
	for _, day := range s.analysisResults {
		for _, r := range day {
			if r.Symbol != symbol {
				continue
			}
			if onlySuccess && !r.Success {
				continue
			}
			if from != nil && r.TradeDate.Before(*from) {
				continue
			}
			if to != nil && r.TradeDate.After(*to) {
				continue
			}
			all = append(all, r)
		}
	}
	// simple sort by date
	sort.Slice(all, func(i, j int) bool {
		return all[i].TradeDate.Before(all[j].TradeDate)
	})
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}

// Get 取得指定日期、指定股票的分析結果。
func (s *Store) Get(ctx context.Context, symbol string, date time.Time) (analysisDomain.DailyAnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dateKey := date.Format("2006-01-02")
	day := s.analysisResults[dateKey]
	for _, r := range day {
		if r.Symbol == symbol {
			return r, nil
		}
	}
	return analysisDomain.DailyAnalysisResult{}, fmt.Errorf("not found")
}

// Helpers to insert stocks and prices
// UpsertTradingPair 建立或回傳既有交易對 ID（以交易對+市場為 key）。
func (s *Store) UpsertTradingPair(pair, name string, market dataDomain.Market, industry string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s-%s", pair, market)
	if id, ok := s.pairByCode[key]; ok {
		return id
	}
	id := s.nextID()
	s.tradingPairs[id] = pairRecord{ID: id, Pair: pair, Name: name, Market: market, Industry: industry, CreatedAt: time.Now()}
	s.pairByCode[key] = id
	return id
}

// InsertDailyPrice 寫入或覆蓋某日單檔的日 K。
func (s *Store) InsertDailyPrice(price dataDomain.DailyPrice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dateKey := price.TradeDate.Format("2006-01-02")
	if _, ok := s.dailyPrices[dateKey]; !ok {
		s.dailyPrices[dateKey] = make(map[string]dataDomain.DailyPrice)
	}
	s.dailyPrices[dateKey][price.Symbol] = price
}

// InsertAnalysisResult 寫入或覆蓋分析結果。
func (s *Store) InsertAnalysisResult(res analysisDomain.DailyAnalysisResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dateKey := res.TradeDate.Format("2006-01-02")
	if _, ok := s.analysisResults[dateKey]; !ok {
		s.analysisResults[dateKey] = make(map[string]analysisDomain.DailyAnalysisResult)
	}
	s.analysisResults[dateKey][res.Symbol] = res
}

// Accessors for prices
// PricesByPair 取得單一交易對的全部日 K 並依日期排序。
func (s *Store) PricesByPair(pair string) []dataDomain.DailyPrice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []dataDomain.DailyPrice
	for _, day := range s.dailyPrices {
		if p, ok := day[pair]; ok {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TradeDate.Before(out[j].TradeDate)
	})
	return out
}

// PricesByDate 取得指定日期的全市場日 K。
func (s *Store) PricesByDate(date time.Time) []dataDomain.DailyPrice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	day := s.dailyPrices[date.Format("2006-01-02")]
	var out []dataDomain.DailyPrice
	for _, p := range day {
		out = append(out, p)
	}
	return out
}

// HasAnalysisForDate 回傳指定交易日是否已有分析結果。
func (s *Store) HasAnalysisForDate(date time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.analysisResults[date.Format("2006-01-02")]) > 0
}

// LatestAnalysisDate 回傳最新的分析日期（成功與否皆考慮）。
func (s *Store) LatestAnalysisDate() (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest time.Time
	for dateKey := range s.analysisResults {
		d, err := time.Parse("2006-01-02", dateKey)
		if err != nil {
			continue
		}
		if d.After(latest) {
			latest = d
		}
	}
	if latest.IsZero() {
		return time.Time{}, false
	}
	return latest, true
}
