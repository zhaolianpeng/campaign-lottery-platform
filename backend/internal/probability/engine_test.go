package probability

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

func TestAliasTable_Basic(t *testing.T) {
	weights := []float64{1, 2, 3, 4}
	table := NewAliasTable(weights)
	if table == nil || table.Len() != 4 {
		t.Fatalf("expected 4 items, got %d", table.Len())
	}
}

func TestAliasTable_Empty(t *testing.T) {
	if NewAliasTable(nil) != nil {
		t.Fatal("expected nil for nil weights")
	}
	if NewAliasTable([]float64{}) != nil {
		t.Fatal("expected nil for empty weights")
	}
	if NewAliasTable([]float64{0, 0, 0}) != nil {
		t.Fatal("expected nil for zero weights")
	}
}

func TestAliasTable_Distribution(t *testing.T) {
	// Single-element table should always return index 0
	table := NewAliasTable([]float64{100})
	for i := 0; i < 100; i++ {
		if idx := table.Next(); idx != 0 {
			t.Fatalf("expected 0, got %d", idx)
		}
	}
}

func TestAliasTable_WeightedDistribution(t *testing.T) {
	// 3 items with very skewed weights: 90, 9, 1
	weights := []float64{90, 9, 1}
	table := NewAliasTable(weights)
	const samples = 1000000
	counts := make([]int, 3)
	for i := 0; i < samples; i++ {
		counts[table.Next()]++
	}
	// Item 0 should be ~90%, Item 2 should be ~1%
	rate0 := float64(counts[0]) / float64(samples)
	rate2 := float64(counts[2]) / float64(samples)
	if rate0 < 0.88 || rate0 > 0.92 {
		t.Errorf("item 0 rate = %.4f, expected ~0.90", rate0)
	}
	if rate2 < 0.005 || rate2 > 0.02 {
		t.Errorf("item 2 rate = %.4f, expected ~0.01", rate2)
	}
}

func TestPityTracker_Basic(t *testing.T) {
	tracker := NewMemoryPityTracker()

	state := tracker.Get("user1", "camp1")
	if state.ConsecutiveMisses != 0 {
		t.Fatalf("expected 0 misses, got %d", state.ConsecutiveMisses)
	}

	state = tracker.IncrementMiss("user1", "camp1")
	if state.ConsecutiveMisses != 1 {
		t.Fatalf("expected 1 miss, got %d", state.ConsecutiveMisses)
	}

	state = tracker.IncrementMiss("user1", "camp1")
	if state.ConsecutiveMisses != 2 {
		t.Fatalf("expected 2 misses, got %d", state.ConsecutiveMisses)
	}

	tracker.Reset("user1", "camp1")
	state = tracker.Get("user1", "camp1")
	if state.ConsecutiveMisses != 0 {
		t.Fatalf("expected 0 after reset, got %d", state.ConsecutiveMisses)
	}
}

func TestCalculateEffectiveProb_NoPity(t *testing.T) {
	cfg := PityConfig{Enabled: false}
	eff := CalculateEffectiveProb(10, cfg, 50)
	if eff != 10 {
		t.Fatalf("expected 10, got %f", eff)
	}
}

func TestCalculateEffectiveProb_UnderThreshold(t *testing.T) {
	cfg := PityConfig{Enabled: true, SoftPityN: 60, PityFactor: 0.015, HardPityN: 90}
	eff := CalculateEffectiveProb(10, cfg, 30)
	if eff != 10 {
		t.Fatalf("expected 10 (under threshold), got %f", eff)
	}
}

func TestCalculateEffectiveProb_SoftPity(t *testing.T) {
	cfg := PityConfig{Enabled: true, SoftPityN: 50, PityFactor: 0.015, HardPityN: 90}
	eff := CalculateEffectiveProb(10, cfg, 50)
	expected := 10 * (1 + 0.015*1) // 10.15
	if math.Abs(eff-expected) > 0.01 {
		t.Fatalf("expected %f, got %f", expected, eff)
	}

	// At 60 misses (10 past threshold)
	eff = CalculateEffectiveProb(10, cfg, 60)
	expected = 10 * (1 + 0.015*11) // 11.65
	if math.Abs(eff-expected) > 0.01 {
		t.Fatalf("expected %f, got %f", expected, eff)
	}
}

func TestCalculateEffectiveProb_HardPity(t *testing.T) {
	cfg := PityConfig{Enabled: true, SoftPityN: 50, PityFactor: 0.015, HardPityN: 90}
	eff := CalculateEffectiveProb(10, cfg, 90)
	if eff != math.MaxFloat64 {
		t.Fatalf("expected MaxFloat64 (hard pity), got %f", eff)
	}
}

func TestEngine_DrawBasic(t *testing.T) {
	weights := []PrizeWeight{
		{ID: "common_1", Weight: 30, Level: "common"},
		{ID: "common_2", Weight: 30, Level: "common"},
		{ID: "rare_1", Weight: 8, Level: "rare"},
		{ID: "secret_1", Weight: 2, Level: "secret"},
	}
	cfg := EngineConfig{
		TargetPrizeID: "secret_1",
		MissWeight:    30,
		Pity: PityConfig{
			Enabled:    true,
			SoftPityN:  20,
			PityFactor: 0.05,
			HardPityN:  50,
		},
	}
	engine := NewEngine(cfg, weights)
	tracker := NewMemoryPityTracker()

	totalDraws := 50000
	prizeCounts := make(map[string]int)
	misCount := 0

	for i := 0; i < totalDraws; i++ {
		result := engine.Draw(cfg.Pity, tracker, "test_user", "test_camp")
		if result.PrizeID == "" {
			misCount++
		} else {
			prizeCounts[result.PrizeID]++
		}
		// Reset after each simulation cycle to keep stats independent
		tracker.Reset("test_user", "test_camp")
	}

	missRate := float64(misCount) / float64(totalDraws) * 100
	t.Logf("Miss rate: %.2f%%", missRate)
	for id, count := range prizeCounts {
		rate := float64(count) / float64(totalDraws) * 100
		t.Logf("  %s: %d (%.2f%%)", id, count, rate)
	}
}

func TestEngine_HardPityGuaranteed(t *testing.T) {
	weights := []PrizeWeight{
		{ID: "secret", Weight: 1, Level: "secret"},
	}
	cfg := EngineConfig{
		TargetPrizeID: "secret",
		MissWeight:    9999, // almost impossible to win normally
		Pity: PityConfig{
			Enabled:    true,
			SoftPityN:  5,
			PityFactor: 0.1,
			HardPityN:  10,
		},
	}
	engine := NewEngine(cfg, weights)
	tracker := NewMemoryPityTracker()

	// Draw repeatedly until hard pity triggers
	for i := 0; i < 20; i++ {
		result := engine.Draw(cfg.Pity, tracker, "user_hard", "camp_hard")
		if i >= 10 {
			// Should have won by hard pity at draw 10 (0-indexed, so 10+ consecutive misses)
			if result.PrizeID == "" {
				t.Fatalf("expected to win by hard pity at draw %d", i)
			}
			if !result.IsHardPity {
				t.Logf("Draw %d: won by non-hard-pity (luck!)", i)
			} else {
				t.Logf("Draw %d: hard pity triggered!", i)
			}
			return
		}
		if result.PrizeID != "" {
			t.Logf("Draw %d: won early (unlikely but possible)", i)
			return // early win resets pity
		}
	}
	t.Fatal("should have triggered hard pity")
}

func TestEngine_Simulation(t *testing.T) {
	weights := []PrizeWeight{
		{ID: "common", Weight: 60, Level: "common"},
		{ID: "rare", Weight: 10, Level: "rare"},
		{ID: "secret", Weight: 1, Level: "secret"},
	}
	cfg := EngineConfig{
		TargetPrizeID: "secret",
		MissWeight:    29,
		Pity: PityConfig{
			Enabled:    true,
			SoftPityN:  30,
			PityFactor: 0.02,
			HardPityN:  60,
		},
	}
	engine := NewEngine(cfg, weights)
	result := engine.Simulate(10000, cfg.Pity)

	t.Logf("\n%s", result.String())

	// Basic sanity checks
	if result.Wins < 5000 {
		t.Errorf("expected >5000 wins, got %d", result.Wins)
	}
	if result.AvgMissesToWin > 50 {
		t.Errorf("avg misses seems too high: %.2f", result.AvgMissesToWin)
	}
}

func TestEngine_MonteCarloDistribution(t *testing.T) {
	// Test without pity — should match the configured weight distribution
	weights := []PrizeWeight{
		{ID: "A", Weight: 50, Level: "common"},
		{ID: "B", Weight: 30, Level: "rare"},
		{ID: "C", Weight: 20, Level: "secret"},
	}
	cfg := EngineConfig{
		TargetPrizeID: "C",
		MissWeight:    0, // always hit
		Pity: PityConfig{
			Enabled: false,
		},
	}
	engine := NewEngine(cfg, weights)
	tracker := NewMemoryPityTracker()

	const samples = 100000
	counts := make(map[string]int)
	for i := 0; i < samples; i++ {
		r := engine.Draw(cfg.Pity, tracker, "mc_user", "mc_camp")
		counts[r.PrizeID]++
	}

	total := counts["A"] + counts["B"] + counts["C"]
	rateA := float64(counts["A"]) / float64(total) * 100
	rateB := float64(counts["B"]) / float64(total) * 100
	rateC := float64(counts["C"]) / float64(total) * 100

	t.Logf("A: %.2f%% (expected 50%%), B: %.2f%% (expected 30%%), C: %.2f%% (expected 20%%)",
		rateA, rateB, rateC)

	if rateA < 47 || rateA > 53 {
		t.Errorf("A rate %.2f%% outside expected range (47-53%%)", rateA)
	}
	if rateC < 17 || rateC > 23 {
		t.Errorf("C rate %.2f%% outside expected range (17-23%%)", rateC)
	}
}

func BenchmarkAliasTable(b *testing.B) {
	rng := rand.New(rand.NewPCG(42, 42))
	weights := make([]float64, 12)
	for i := range weights {
		weights[i] = rng.Float64()*100 + 1
	}
	table := NewAliasTable(weights)
	if table == nil {
		b.Fatal("failed to build alias table")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.Next()
	}
}

func BenchmarkEngineDraw(b *testing.B) {
	weights := make([]PrizeWeight, 12)
	for i := range weights {
		weights[i] = PrizeWeight{
			ID:     fmt.Sprintf("prize_%d", i),
			Weight: float64(100 - i*5),
			Level:  "common",
		}
	}
	cfg := EngineConfig{
		TargetPrizeID: "prize_0",
		MissWeight:    50,
		Pity: PityConfig{
			Enabled:    true,
			SoftPityN:  30,
			PityFactor: 0.02,
			HardPityN:  60,
		},
	}
	engine := NewEngine(cfg, weights)
	tracker := NewMemoryPityTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Draw(cfg.Pity, tracker, "bench_user", "bench_camp")
	}
}
