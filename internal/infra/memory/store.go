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
)

// Store 為 MVP 使用的記憶體資料庫，僅供示範且非併發安全。
type Store struct {
	mu              sync.RWMutex
	users           map[string]authDomain.User
	passwords       map[string]string
	tokens          map[string]tokenRecord
	stocks          map[string]stockRecord                                   // id -> record
	stockByCode     map[string]string                                        // code+market -> id
	dailyPrices     map[string]map[string]dataDomain.DailyPrice              // date -> stockID -> price
	analysisResults map[string]map[string]analysisDomain.DailyAnalysisResult // date -> stockID -> result
	idSeq           int64
}

type tokenRecord struct {
	UserID  string
	Expires time.Time
}

type stockRecord struct {
	ID        string
	Code      string
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
		stocks:          make(map[string]stockRecord),
		stockByCode:     make(map[string]string),
		dailyPrices:     make(map[string]map[string]dataDomain.DailyPrice),
		analysisResults: make(map[string]map[string]analysisDomain.DailyAnalysisResult),
	}
}

// ID generator (simple incremental).
func (s *Store) nextID() string {
	s.idSeq++
	return fmt.Sprintf("id-%d", s.idSeq)
}

// SeedUsers 建立預設帳號供登入測試。
func (s *Store) SeedUsers() {
	s.addUser("admin@example.com", "admin", "Admin", authDomain.RoleAdmin)
	s.addUser("analyst@example.com", "analyst", "Analyst", authDomain.RoleAnalyst)
	s.addUser("user@example.com", "password", "User", authDomain.RoleUser)
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

func (m *MemoryTokenIssuer) Issue(ctx context.Context, user authDomain.User) (authDomain.TokenPair, error) {
	token := fmt.Sprintf("token-%s-%d", user.ID, time.Now().UnixNano())
	m.store.mu.Lock()
	m.store.tokens[token] = tokenRecord{
		UserID:  user.ID,
		Expires: time.Now().Add(m.ttl),
	}
	m.store.mu.Unlock()
	return authDomain.TokenPair{
		AccessToken:   token,
		RefreshToken:  "",
		AccessExpiry:  time.Now().Add(m.ttl),
		RefreshExpiry: time.Now().Add(m.ttl),
	}, nil
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
// UpsertStock 建立或回傳既有股票 ID（以代碼+市場為 key）。
func (s *Store) UpsertStock(code, name string, market dataDomain.Market, industry string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s-%s", code, market)
	if id, ok := s.stockByCode[key]; ok {
		return id
	}
	id := s.nextID()
	s.stocks[id] = stockRecord{ID: id, Code: code, Name: name, Market: market, Industry: industry, CreatedAt: time.Now()}
	s.stockByCode[key] = id
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
// PricesBySymbol 取得單檔股票的全部日 K 並依日期排序。
func (s *Store) PricesBySymbol(symbol string) []dataDomain.DailyPrice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []dataDomain.DailyPrice
	for _, day := range s.dailyPrices {
		if p, ok := day[symbol]; ok {
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
