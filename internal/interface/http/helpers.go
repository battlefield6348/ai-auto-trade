package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-auto-trade/internal/application/trading"
	tradingDomain "ai-auto-trade/internal/domain/trading"

	"github.com/gin-gonic/gin"
)

func (s *Server) getSymbol(c *gin.Context) string {
	return strings.ToUpper(strings.TrimSpace(c.DefaultQuery("symbol", "BTCUSDT")))
}

func (s *Server) getTimeframe(c *gin.Context, def string) string {
	return c.DefaultQuery("timeframe", def)
}

func (s *Server) parseDateRange(c *gin.Context) (time.Time, time.Time, error) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	if startStr == "" {
		// Default to last 30 days
		end := time.Now()
		start := end.AddDate(0, 0, -30)
		return start, end, nil
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date")
	}

	var end time.Time
	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date")
		}
	}

	return start, end, nil
}

func (s *Server) setRefreshCookie(c *gin.Context, token string, expiry time.Time) {
	host, _, _ := strings.Cut(c.Request.Host, ":")
	isLocal := host == "localhost" || host == "127.0.0.1"

	c.SetCookie(
		refreshCookieName,
		token,
		int(time.Until(expiry).Seconds()),
		"/",
		"",
		!isLocal, // Secure: only if not local
		true,     // HttpOnly
	)
}

func errorString(err error) interface{} {
	if err == nil {
		return nil
	}
	return err.Error()
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func optionalString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func taipeiLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		return time.FixedZone("Asia/Taipei", 8*3600)
	}
	return loc
}

func clientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		host, _, _ := strings.Cut(r.RemoteAddr, ":")
		ip = host
	}
	return strings.TrimSpace(strings.Split(ip, ",")[0])
}

func parseBearer(h string) string {
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func currentUserID(c *gin.Context) string {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func parseStrategyPath(path string) (string, string) {
	const prefix = "/api/admin/strategies/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(trimmed, "/", 2)
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action
}

func parseIntDefault(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func parseFloatDefault(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseBoolDefault(s string, def bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func formatPercent(v float64) string {
	return fmt.Sprintf("%.2f%%", v*100)
}

func formatOptionalPercent(v *float64) string {
	if v == nil {
		return "N/A"
	}
	return formatPercent(*v)
}

func formatOptionalTimes(v *float64) string {
	if v == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.2fx", *v)
}

func (s *Server) recordJob(j jobRun) {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if j.DataSource == "" {
		j.DataSource = s.dataSource
	}
	s.jobHistory = append(s.jobHistory, j)
	if len(s.jobHistory) > 50 {
		s.jobHistory = s.jobHistory[len(s.jobHistory)-50:]
	}
	if j.Kind == "auto" {
		s.lastAutoRun = j.End
	}
}

func jobRunToMap(j jobRun, loc *time.Location) map[string]interface{} {
	start := j.Start.In(loc)
	end := j.End.In(loc)
	duration := int(end.Sub(start).Seconds())
	return map[string]interface{}{
		"kind":             j.Kind,
		"triggered_by":     optionalString(j.TriggeredBy),
		"start":            start.Format(time.RFC3339),
		"end":              end.Format(time.RFC3339),
		"duration_seconds": duration,
		"data_source":      optionalString(j.DataSource),
		"ingestion": map[string]interface{}{
			"success": j.IngestionOK,
			"error":   optionalString(j.IngestionErr),
		},
		"analysis": map[string]interface{}{
			"enabled":       j.AnalysisOn,
			"success":       j.AnalysisOK,
			"total":         j.AnalysisTotal,
			"success_count": j.AnalysisSucc,
			"failure_count": j.AnalysisFail,
			"error":         optionalString(j.AnalysisErr),
		},
		"failures": j.Failures,
	}
}

func buildBacktestInput(body strategyBacktestRequest, strategyID string, inline *tradingDomain.Strategy) (trading.BacktestInput, error) {
	var input trading.BacktestInput
	if body.StartDate == "" || body.EndDate == "" {
		return input, fmt.Errorf("start_date and end_date required")
	}
	start, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		return input, fmt.Errorf("invalid start_date")
	}
	end, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		return input, fmt.Errorf("invalid end_date")
	}
	pm := tradingDomain.PriceMode(body.PriceMode)
	if pm == "" {
		pm = tradingDomain.PriceNextOpen
	}
	
	input = trading.BacktestInput{
		StrategyID:      strategyID,
		Inline:          inline,
		StartDate:       start,
		EndDate:         end,
		InitialEquity:   body.InitialEquity,
		PriceMode:       &pm,
		StopLossPct:     body.StopLossPct,
		TakeProfitPct:   body.TakeProfitPct,
		MaxDailyLossPct: body.MaxDailyLossPct,
	}
	
	if body.CoolDownDays != 0 {
		days := body.CoolDownDays
		input.CoolDownDays = &days
	}
	if body.MinHoldDays != 0 {
		days := body.MinHoldDays
		input.MinHoldDays = &days
	}
	if body.MaxPositions != 0 {
		p := body.MaxPositions
		input.MaxPositions = &p
	}
	if body.FeesPct != 0 {
		f := body.FeesPct
		input.FeesPct = &f
	}
	if body.SlippagePct != 0 {
		s := body.SlippagePct
		input.SlippagePct = &s
	}
	
	return input, nil
}
