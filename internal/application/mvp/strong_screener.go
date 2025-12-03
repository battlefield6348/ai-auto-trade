package mvp

import (
	"context"
	"sort"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
)

// StrongScreener 提供 MVP 固定條件的強勢股篩選。
// 條件：
// - score >= score_min
// - return_5d > 0
// - volume_ratio >= volume_ratio_min
// - change_percent >= 0
type StrongScreener struct {
	queryUsecase *analysis.QueryUseCase
}

// NewStrongScreener 建立強勢股篩選器，使用固定條件執行 MVP 選股。
func NewStrongScreener(repo analysis.AnalysisQueryRepository) *StrongScreener {
	return &StrongScreener{
		queryUsecase: analysis.NewQueryUseCase(repo),
	}
}

type StrongScreenerInput struct {
	TradeDate      time.Time
	Limit          int
	ScoreMin       float64
	VolumeRatioMin float64
}

type StrongScreenerResult struct {
	TradeDate  time.Time
	TotalCount int
	Items      []analysisDomain.DailyAnalysisResult
}

func (s *StrongScreener) Run(ctx context.Context, input StrongScreenerInput) (StrongScreenerResult, error) {
	var out StrongScreenerResult

	if input.TradeDate.IsZero() {
		return out, nil
	}
	if input.Limit <= 0 {
		input.Limit = 50
	}
	if input.ScoreMin == 0 {
		input.ScoreMin = 70
	}
	if input.VolumeRatioMin == 0 {
		input.VolumeRatioMin = 1.5
	}

	resp, err := s.queryUsecase.QueryByDate(ctx, analysis.QueryByDateInput{
		Date: input.TradeDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Pagination: analysis.Pagination{
			Offset: 0,
			Limit:  10000,
		},
	})
	if err != nil {
		return out, err
	}

	out.TradeDate = input.TradeDate
	filtered := make([]analysisDomain.DailyAnalysisResult, 0, len(resp.Results))

	for _, r := range resp.Results {
		if r.Score < input.ScoreMin {
			continue
		}
		if r.Return5 == nil || *r.Return5 <= 0 {
			continue
		}
		if r.VolumeMultiple == nil || *r.VolumeMultiple < input.VolumeRatioMin {
			continue
		}
		if r.ChangeRate < 0 {
			continue
		}
		filtered = append(filtered, r)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Score == filtered[j].Score {
			ri := 0.0
			rj := 0.0
			if filtered[i].Return5 != nil {
				ri = *filtered[i].Return5
			}
			if filtered[j].Return5 != nil {
				rj = *filtered[j].Return5
			}
			return ri > rj
		}
		return filtered[i].Score > filtered[j].Score
	})

	out.TotalCount = len(filtered)
	if len(filtered) > input.Limit {
		filtered = filtered[:input.Limit]
	}
	out.Items = filtered
	return out, nil
}
