package analysis

import (
	"fmt"
	"time"

	"ai-auto-trade/internal/domain/dataingestion"
)

// Tag 表示分析結果的標籤。
type Tag string

const (
	TagShortTermStrong Tag = "短期強勢"
	TagVolumeSurge     Tag = "量能放大"
	TagNearHigh        Tag = "接近前高"
	TagNearLow         Tag = "接近前低"
	TagHighVolatility  Tag = "高波動"
	TagLowVolatility   Tag = "低波動"
)

// DailyAnalysisResult 為「股票 × 日期」的分析結果。
type DailyAnalysisResult struct {
	Symbol      string
	Market      dataingestion.Market
	Industry    string
	TradeDate   time.Time
	Version     string

	// 價格／報酬
	Close       float64
	Change      float64
	ChangeRate  float64
	Return5     *float64
	Return20    *float64
	Return60    *float64
	High20      *float64
	Low20       *float64
	RangePos20  *float64 // 0~1 區間位置

	// 均線
	MA5         *float64
	MA10        *float64
	MA20        *float64
	MA60        *float64
	Deviation20 *float64 // 收盤價相對 MA20 的乖離率

	// 成交量
	Volume          int64
	AvgVolume5      *float64
	AvgVolume20     *float64
	VolumeMultiple  *float64 // 當日量 / 近 20 日均量

	// 波動度
	Amplitude        *float64
	AvgAmplitude20   *float64

	// 分數與標籤
	Score   float64
	Tags    []Tag

	// 狀態
	Success     bool
	ErrorReason string
}

// Validate 基礎必填檢查。
func (r DailyAnalysisResult) Validate() error {
	if r.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	if r.TradeDate.IsZero() {
		return fmt.Errorf("trade date is required")
	}
	switch r.Market {
	case dataingestion.MarketTWSE, dataingestion.MarketTPEx:
	default:
		return fmt.Errorf("market is required or unsupported")
	}
	return nil
}
