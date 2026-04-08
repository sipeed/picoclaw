package main

import (
	"math"
	"testing"
)

func TestComputeModeAggAllCategories(t *testing.T) {
	results := []EvalResult{
		{
			Mode:     "test",
			SampleID: "s1",
			QAResults: []QAResult{
				{Category: 1, TokenF1: 0.5, HitRate: 0.8},
				{Category: 2, TokenF1: 0.3, HitRate: 0.6},
				{Category: 3, TokenF1: 0.1, HitRate: 0.4},
				{Category: 4, TokenF1: 0.7, HitRate: 0.9},
				{Category: 5, TokenF1: 0.2, HitRate: 0.1},
			},
		},
	}
	for i := range results {
		results[i].Agg = aggregateMetrics(results[i].QAResults)
	}

	got := computeModeAgg(results)

	// Should have all 5 categories
	for cat := 1; cat <= 5; cat++ {
		cm, ok := got.ByCategory[cat]
		if !ok {
			t.Errorf("ByCategory missing category %d", cat)
			continue
		}
		if cm.QuestionCount != 1 {
			t.Errorf("ByCategory[%d].QuestionCount = %d, want 1", cat, cm.QuestionCount)
		}
	}

	// Verify specific F1 values per category
	wantF1 := map[int]float64{1: 0.5, 2: 0.3, 3: 0.1, 4: 0.7, 5: 0.2}
	for cat, want := range wantF1 {
		if cm, ok := got.ByCategory[cat]; ok {
			if math.Abs(cm.F1-want) > 1e-9 {
				t.Errorf("ByCategory[%d].F1 = %.4f, want %.4f", cat, cm.F1, want)
			}
		}
	}
}

func TestComputeModeAgg(t *testing.T) {
	// Two samples with different question counts:
	//   sample-a: 2 questions, F1 = [0.4, 0.6] → avg 0.5
	//   sample-b: 8 questions, F1 = [0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1] → avg 0.1
	//
	// Unweighted (PrintComparison bug): (0.5 + 0.1) / 2 = 0.3
	// Weighted (correct):              (0.4+0.6 + 0.1*8) / 10 = 1.8 / 10 = 0.18
	results := []EvalResult{
		{
			Mode:     "test",
			SampleID: "sample-a",
			QAResults: []QAResult{
				{TokenF1: 0.4, HitRate: 0.5},
				{TokenF1: 0.6, HitRate: 0.7},
			},
		},
		{
			Mode:     "test",
			SampleID: "sample-b",
			QAResults: []QAResult{
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
				{TokenF1: 0.1, HitRate: 0.2},
			},
		},
	}
	// Compute per-sample aggregates
	for i := range results {
		results[i].Agg = aggregateMetrics(results[i].QAResults)
	}

	got := computeModeAgg(results)

	// Weighted: (0.4+0.6+0.1*8) / 10 = 1.8/10 = 0.18
	wantF1 := 0.18
	if math.Abs(got.OverallF1-wantF1) > 1e-9 {
		t.Errorf("OverallF1 = %.6f, want %.6f (weighted average)", got.OverallF1, wantF1)
	}

	// Weighted: (0.5+0.7+0.2*8) / 10 = 2.8/10 = 0.28
	wantRecall := 0.28
	if math.Abs(got.OverallHitRate-wantRecall) > 1e-9 {
		t.Errorf("OverallHitRate = %.6f, want %.6f (weighted average)", got.OverallHitRate, wantRecall)
	}

	if got.TotalQuestions != 10 {
		t.Errorf("TotalQuestions = %d, want 10", got.TotalQuestions)
	}
}
