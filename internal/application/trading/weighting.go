package trading

// WeightedCondition 定義單一條件的權重。
type WeightedCondition struct {
    Key    string
    Weight float64
}

// ComputeWeightedScore 根據條件達成與否計算加權總分。
// truth 內的 key 為條件名稱，value 為是否達成；未出現在 truth 的視為 false。
// 範例：如果條件 A 達成且權重 0.6，則貢獻 0.6；未達成則 0。
func ComputeWeightedScore(truth map[string]bool, weights []WeightedCondition) float64 {
    var score float64
    for _, wc := range weights {
        if wc.Weight <= 0 {
            continue
        }
        if truth[wc.Key] {
            score += wc.Weight
        }
    }
    return score
}

// ReachThreshold 回傳加權分數是否達到目標，並回傳分數。
func ReachThreshold(truth map[string]bool, weights []WeightedCondition, threshold float64) (float64, bool) {
    score := ComputeWeightedScore(truth, weights)
    return score, score >= threshold
}
