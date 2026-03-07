package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"ai-auto-trade/internal/application/strategy"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

type optimizerResult struct {
	EntryThreshold float64
	ExitThreshold  float64
	TakeProfit     float64
	StopLoss       float64
	TotalReturn    float64
	WinRate        float64
	TotalTrades    int
}

func floatPtr(v float64) *float64 { return &v }

func buildRules(changeMin, volMin, maMin, rangeMin float64, ruleType string) []strategyDomain.StrategyRule {
	var rules []strategyDomain.StrategyRule

	// Base Score Weight
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   50.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type: "BASE_SCORE",
		},
	})

	// Price Return
	p1, _ := json.Marshal(map[string]interface{}{"days": 1, "min": changeMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   20.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "PRICE_RETURN",
			ParamsRaw: p1,
		},
	})

	// Volume Surge
	p2, _ := json.Marshal(map[string]interface{}{"min": volMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   15.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "VOLUME_SURGE",
			ParamsRaw: p2,
		},
	})

	// MA Deviation
	p3, _ := json.Marshal(map[string]interface{}{"ma": 20, "min": maMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   15.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "MA_DEVIATION",
			ParamsRaw: p3,
		},
	})

	// Range Pos
	p4, _ := json.Marshal(map[string]interface{}{"days": 20, "min": rangeMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   10.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "RANGE_POS",
			ParamsRaw: p4,
		},
	})

	return rules
}

// Custom data provider that just passes the history inside memory
type memDataProvider struct {
	history []analysisDomain.DailyAnalysisResult
}

func (m *memDataProvider) FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	var filtered []analysisDomain.DailyAnalysisResult
	for _, h := range m.history {
		if from != nil && h.TradeDate.Before(*from) {
			continue
		}
		if to != nil && h.TradeDate.After(*to) {
			continue
		}
		filtered = append(filtered, h)
	}
	return filtered, nil
}

func fetchBinanceData(symbol string, limit int) []analysisDomain.DailyAnalysisResult {
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=1d&limit=%d", symbol, limit)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var rawData [][]interface{}
	json.Unmarshal(body, &rawData)

	// Open Time 0, Open 1, High 2, Low 3, Close 4, Volume 5
	var items []analysisDomain.DailyAnalysisResult
	
	// Create raw data array
	type Kline struct {
		T time.Time
		O, H, L, C float64
		V float64
	}
	klines := make([]Kline, len(rawData))

	for i, k := range rawData {
		ts := int64(k[0].(float64))
		tradeTime := time.UnixMilli(ts)

		o, _ := strconv.ParseFloat(k[1].(string), 64)
		h, _ := strconv.ParseFloat(k[2].(string), 64)
		l, _ := strconv.ParseFloat(k[3].(string), 64)
		c, _ := strconv.ParseFloat(k[4].(string), 64)
		v, _ := strconv.ParseFloat(k[5].(string), 64)

		klines[i] = Kline{tradeTime, o, h, l, c, v}
	}

	for i, k := range klines {
		if i < 20 {
			// skip first 20 days since they don't have MA20
			continue
		}

		// Calculate MA20
		sumClose := 0.0
		sumVol := 0.0
		highest := 0.0
		lowest := 9999999999.0
		
		for j := i - 19; j <= i; j++ {
			sumClose += klines[j].C
			if klines[j].H > highest { highest = klines[j].H }
			if klines[j].L < lowest { lowest = klines[j].L }
		}
		ma20 := sumClose / 20.0
		for j := i - 5; j <= i; j++ { // 5-day average volume for VolumeRatio
			sumVol += klines[j].V
		}
		avgVol5 := sumVol / 5.0
		
		volRatio := 0.0
		if avgVol5 > 0 {
			volRatio = k.V / avgVol5
		}

		changeRate := 0.0
		if klines[i-1].C > 0 {
			changeRate = (k.C / klines[i-1].C - 1.0) * 100 // %
		}

		pricePos20 := 0.0
		if highest > lowest {
			pricePos20 = (k.C - lowest) / (highest - lowest) * 100
		}
		
		// Base score (simulate a simple score for backtest)
		baseScore := 60.0
		if changeRate > 0 { baseScore += 10 }
		if k.C > ma20 { baseScore += 15 }
		
		items = append(items, analysisDomain.DailyAnalysisResult{
			Symbol:      symbol,
			TradeDate:   k.T,
			Close:       k.C,
			ChangeRate:  changeRate,
			VolumeMultiple: floatPtr(volRatio),
			MA20:        floatPtr(ma20),
			RangePos20:  floatPtr(pricePos20 / 100.0), // Need to match percentage format 0.0-1.0
			Score:       baseScore,
			// For Return5 we mock it if it's past, but Backtest doesn't strictly need it for strategy execution
		})
	}
	return items
}

func main() {
	symbol := "BTCUSDT"
	log.Println("Fetching last 300 days data from Binance to accommodate MA calculations...")
	history := fetchBinanceData(symbol, 300) // About 280 days of usable data
	
	if len(history) == 0 {
		log.Fatalf("No data fetched")
	}

	startDate := history[0].TradeDate
	endDate := history[len(history)-1].TradeDate
	log.Printf("Data range available from %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	dp := &memDataProvider{history: history}
	uc := strategy.NewBacktestUseCase(nil, dp)

	horizons := []int{3, 5, 10}

	var results []optimizerResult

	// Configurable params to iterate
	entryThresholds := []float64{50, 60, 70, 75}
	exitThresholds := []float64{30, 40, 50, 60}
	takeProfits := []float64{0.05, 0.08, 0.12, 0.20}
	stopLosses := []float64{-0.02, -0.05, -0.10}
	
	// Bonus condition thresholds to iterate
	changeMins := []float64{0.5, 1.0, 2.0}
	volMins := []float64{1.2, 1.5}

	log.Println("Starting parameter permutation...")

	for _, eth := range entryThresholds {
		for _, xth := range exitThresholds {
			for _, tp := range takeProfits {
				for _, sl := range stopLosses {
					for _, cm := range changeMins {
						for _, vm := range volMins {
						
							strat := &strategyDomain.ScoringStrategy{
								Name:          "Optimizer Strategy",
								Timeframe:     "1d",
								BaseSymbol:    symbol,
								Threshold:     eth,
								ExitThreshold: xth,
							}
							strat.EntryRules = buildRules(cm, vm, 1.0, 0.8, "entry")
							strat.ExitRules = buildRules(-cm*1.5, 0.8, -0.5, 0.8, "exit")
							strat.Risk = tradingDomain.RiskSettings{
								StopLossPct:   floatPtr(sl),
								TakeProfitPct: floatPtr(tp),
							}

							res, err := uc.ExecuteWithStrategy(context.Background(), strat, symbol, startDate, endDate, horizons)
							if err != nil {
								continue
							}

							if res != nil && res.Summary.TotalTrades > 0 {
								results = append(results, optimizerResult{
									EntryThreshold: eth,
									ExitThreshold:  xth,
									TakeProfit:     tp,
									StopLoss:       sl,
									TotalReturn:    res.Summary.TotalReturn,
									WinRate:        res.Summary.WinRate,
									TotalTrades:    res.Summary.TotalTrades,
								})
							}
							
						}
					}
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalReturn > results[j].TotalReturn // Descending
	})

	fmt.Println("\n=== Top 5 Short-Term Optimal Strategies (Past 3 Months) ===")
	if len(results) == 0 {
		fmt.Println("No profitable generic strategies found.")
	}
	
	for i := 0; i < 5 && i < len(results); i++ {
		r := results[i]
		fmt.Printf("#%d | Total Return: %6.2f%% | WinRate: %5.2f%% | Trades: %d \n", i+1, r.TotalReturn, r.WinRate, r.TotalTrades)
		fmt.Printf("   -> Params: EntryScore: %.0f, ExitScore: %.0f, TP: %.0f%%, SL: %.0f%%\n\n", 
			r.EntryThreshold, r.ExitThreshold, r.TakeProfit*100, r.StopLoss*100)
	}
}
