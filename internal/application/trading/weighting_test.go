package trading

import "testing"

func almostEqual(a, b, eps float64) bool {
    if a > b {
        return a-b < eps
    }
    return b-a < eps
}

func TestComputeWeightedScore(t *testing.T) {
    weights := []WeightedCondition{
        {Key: "condA", Weight: 0.6},
        {Key: "condB", Weight: 0.2},
        {Key: "condC", Weight: 0.3},
    }
    truth := map[string]bool{
        "condA": true,
        "condB": false,
        "condC": true,
    }
    score := ComputeWeightedScore(truth, weights)
    if !almostEqual(score, 0.9, 1e-9) {
        t.Fatalf("expected 0.9, got %v", score)
    }
    score, ok := ReachThreshold(truth, weights, 0.7)
    if !ok {
        t.Fatalf("expected threshold reached, score=%v", score)
    }
    _, ok = ReachThreshold(truth, weights, 1.0)
    if ok {
        t.Fatalf("expected threshold not reached")
    }
}
