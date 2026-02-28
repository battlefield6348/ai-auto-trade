package memory

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/auth"
	"ai-auto-trade/internal/domain/dataingestion"
)

func TestStore_Users(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	t.Run("CreateAndFind", func(t *testing.T) {
		id, err := s.Create(ctx, auth.User{Email: "test@example.com", Name: "Test"})
		if err != nil {
			t.Fatal(err)
		}
		u, err := s.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if u.Email != "test@example.com" {
			t.Errorf("expected email mismatch: %s", u.Email)
		}
		
		u2, err := s.FindByEmail(ctx, "test@example.com")
		if err != nil || u2.ID != id {
			t.Error("FindByEmail failed")
		}
	})

	t.Run("SeedUsers", func(t *testing.T) {
		s.SeedUsers()
		_, err := s.FindByEmail(ctx, "admin@example.com")
		if err != nil {
			t.Error("admin user seed failed")
		}
	})
}

func TestStore_DataIngestion(t *testing.T) {
	s := NewStore()
	
	t.Run("UpsertTradingPair", func(t *testing.T) {
		id := s.UpsertTradingPair("BTCUSDT", "Bitcoin", dataingestion.MarketCrypto, "Finance")
		if id == "" {
			t.Error("id is empty")
		}
	})

	t.Run("InsertDailyPrice", func(t *testing.T) {
		now := time.Now()
		p := dataingestion.DailyPrice{
			Symbol:    "BTCUSDT",
			TradeDate: now,
			Close:     50000,
		}
		s.InsertDailyPrice(p)
		
		prices := s.PricesByDate(now)
		if len(prices) != 1 {
			t.Errorf("expected 1 price, got %d", len(prices))
		}
		
		pricesByPair := s.PricesByPair("BTCUSDT")
		if len(pricesByPair) != 1 {
			t.Error("PricesByPair failed")
		}
	})
}

func TestStore_Sessions(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	t.Run("SaveAndGetSession", func(t *testing.T) {
		sess := auth.Session{
			Token:  "token-1",
			UserID: "u1",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		err := s.SaveSession(ctx, sess)
		if err != nil {
			t.Fatal(err)
		}
		
		got, err := s.GetSession(ctx, "token-1")
		if err != nil || got.UserID != "u1" {
			t.Error("GetSession failed")
		}
		
		err = s.RevokeSession(ctx, "token-1")
		if err != nil {
			t.Error("RevokeSession failed")
		}
	})
}

func TestMemoryTokenIssuer(t *testing.T) {
	s := NewStore()
	issuer := NewMemoryTokenIssuer(s, time.Hour)
	ctx := context.Background()

	t.Run("IssueAndValidate", func(t *testing.T) {
		user := auth.User{ID: "u1", Email: "u1@test.com"}
		s.addUser(user.Email, "pass", "User 1", auth.RoleUser)
		// addUser generates a new ID, so we should find it
		user, _ = s.FindByEmail(ctx, user.Email)

		pair, err := issuer.Issue(ctx, user, auth.TokenMeta{})
		if err != nil {
			t.Fatal(err)
		}
		
		u, ok := s.ValidateToken(pair.AccessToken)
		if !ok || u.Email != "u1@test.com" {
			t.Errorf("ValidateToken failed, ok=%v", ok)
		}
		
		// For Refresh to work in memory, we need a session
		s.SaveSession(ctx, auth.Session{
			Token:     pair.RefreshToken,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(time.Hour),
		})

		newPair, err := issuer.Refresh(ctx, pair.RefreshToken)
		if err != nil {
			t.Errorf("Refresh failed: %v", err)
		}
		if newPair.AccessToken == pair.AccessToken {
			t.Error("expected new access token")
		}
		
		err = issuer.RevokeRefresh(ctx, newPair.RefreshToken)
		if err != nil {
			t.Error("RevokeRefresh failed")
		}
	})
}

func TestStore_Analysis(t *testing.T) {
	s := NewStore()
	ctx := context.Background()
	now := time.Now()

	t.Run("AnalysisOps", func(t *testing.T) {
		res := analysisDomain.DailyAnalysisResult{
			Symbol:    "BTCUSDT",
			TradeDate: now,
			Close:     60000,
		}
		s.InsertAnalysisResult(res)
		
		if !s.HasAnalysisForDate(now) {
			t.Error("expected analysis to exist")
		}
		
		d, ok := s.LatestAnalysisDate()
		if !ok || d.IsZero() {
			t.Error("LatestAnalysisDate failed")
		}
		
		got, err := s.Get(ctx, "BTCUSDT", now)
		if err != nil || got.Close != 60000 {
			t.Error("Get failed")
		}
		
		history, err := s.FindHistory(ctx, "BTCUSDT", &now, &now, 10, false)
		if err != nil || len(history) != 1 {
			t.Error("FindHistory failed")
		}
		
		results, total, err := s.FindByDate(ctx, now, analysis.QueryFilter{}, analysis.SortOption{}, analysis.Pagination{Limit: 10})
		if err != nil || total != 1 || len(results) != 1 {
			t.Error("FindByDate failed")
		}
	})
}

func TestStore_Presets(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	t.Run("SaveAndLoad", func(t *testing.T) {
		err := s.Save(ctx, "u1", []byte("config"))
		if err != nil {
			t.Fatal(err)
		}
		
		data, err := s.Load(ctx, "u1")
		if err != nil || string(data) != "config" {
			t.Error("Load failed")
		}
		
		_, err = s.Load(ctx, "u2")
		if !IsPresetNotFound(err) {
			t.Error("expected not found error")
		}
	})
}

func TestOwnerChecker(t *testing.T) {
	oc := OwnerChecker{}
	if !oc.IsOwner(context.Background(), "u1", "u1") {
		t.Error("IsOwner failed")
	}
}
