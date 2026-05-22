// Package probability implements a configurable lottery draw engine with:
//   - Weighted random selection (Alias Method, O(1) per draw)
//   - Soft pity (increasing probability with consecutive misses)
//   - Hard pity (guaranteed hit after N misses)
//   - Monte Carlo verification utilities
package probability

import (
	"fmt"
	"math"
	"sort"
)

// PrizeWeight defines a single prize's weight for the draw engine.
type PrizeWeight struct {
	ID     string
	Weight float64 // 绝对概率权重（可被 pity 提升）
	Level  string  // common / rare / secret / limited
}

// DrawResult represents the result of a single draw operation.
type DrawResult struct {
	PrizeID           string  // 中奖奖品ID（空=未中奖）
	PrizeLevel        string  // 奖品等级
	ConsecutiveMisses int     // 当前连续未中次数
	PityMultiplier    float64 // 当前 pity 概率倍数
	IsHardPity        bool    // 是否硬保底触发

	// 🆕 UP池相关
	IsUPPoolWin bool // 是否是UP池中奖
}

// EngineConfig defines the overall configuration for a draw engine instance.
type EngineConfig struct {
	TargetPrizeID string     // 保底目标奖品ID（通常是隐藏款）
	Pity          PityConfig // 保底配置
	MissWeight    float64    // 未中奖的权重（"谢谢参与"）
}

// Engine is the main lottery draw engine.
// It is NOT safe for concurrent use — create one per campaign or use external synchronization.
type Engine struct {
	config       EngineConfig
	weights      map[string]float64 // prize ID -> base probability weight
	prizeLevels  map[string]string  // prize ID -> level
	aliasTable   *AliasTable
	prizeIDs     []string
	aliasReady   bool
}

// NewEngine creates a new probability engine with the given config and prize weights.
func NewEngine(cfg EngineConfig, prizeWeights []PrizeWeight) *Engine {
	weights := make(map[string]float64, len(prizeWeights))
	levels := make(map[string]string, len(prizeWeights))
	ids := make([]string, 0, len(prizeWeights))

	for _, pw := range prizeWeights {
		weights[pw.ID] = pw.Weight
		levels[pw.ID] = pw.Level
		ids = append(ids, pw.ID)
	}

	// Sort for deterministic ordering
	sort.Strings(ids)

	e := &Engine{
		config:      cfg,
		weights:     weights,
		prizeLevels: levels,
		prizeIDs:    ids,
	}

	e.rebuildAlias(weights, ids)
	return e
}

// rebuildAlias rebuilds the alias table from the current weight map.
func (e *Engine) rebuildAlias(weights map[string]float64, ids []string) {
	weightList := make([]float64, 0, len(ids))
	for _, id := range ids {
		weightList = append(weightList, weights[id])
	}
	e.aliasTable = NewAliasTable(weightList)
	e.aliasReady = e.aliasTable != nil
}

// Draw performs a single weighted random draw with pity considerations.
// Returns the draw result and any error.
//
// The algorithm works as follows:
//  1. Determine if pity has elevated the target prize to guarantee level.
//  2. If hard pity triggers, return the target prize directly.
//  3. Otherwise, build dynamic weights considering soft pity multiplier,
//     then perform a weighted random selection using the Alias Method.
func (e *Engine) Draw(cfg PityConfig, tracker PityTracker, userID, campaignID string) DrawResult {
	result := DrawResult{}

	state := tracker.Get(userID, campaignID)
	n := state.ConsecutiveMisses
	result.ConsecutiveMisses = n

	// Build dynamic weight map accounting for pity
	dynamicWeights := make(map[string]float64)
	totalWeight := e.config.MissWeight

	for _, id := range e.prizeIDs {
		baseWeight := e.weights[id]
		dynamicWeights[id] = baseWeight
		totalWeight += baseWeight
	}

	// Apply soft pity to the target prize
	if cfg.Enabled && n >= cfg.SoftPityN && cfg.TargetWeight > 0 {
		effectiveWeight := CalculateEffectiveProb(cfg.TargetWeight, cfg, n)
		dynamicWeights[e.config.TargetPrizeID] = effectiveWeight

		// Recalculate total
		totalWeight = e.config.MissWeight
		for _, w := range dynamicWeights {
			totalWeight += w
		}

		result.PityMultiplier = effectiveWeight / cfg.TargetWeight

		// Hard pity check
		if cfg.HardPityN > 0 && n >= cfg.HardPityN {
			result.PrizeID = e.config.TargetPrizeID
			result.PrizeLevel = e.prizeLevels[e.config.TargetPrizeID]
			result.IsHardPity = true
			result.PityMultiplier = math.MaxFloat64
			return result
		}
	} else {
		result.PityMultiplier = 1.0
	}

	// Build weighted list for alias table (include miss as an option if MissWeight > 0)
	samplerWeights := make([]float64, 0, len(dynamicWeights)+1)
	samplerIDs := make([]string, 0, len(dynamicWeights)+1)

	// Add "miss" as entry 0 if configured
	if e.config.MissWeight > 0 {
		samplerWeights = append(samplerWeights, e.config.MissWeight)
		samplerIDs = append(samplerIDs, "")
	}

	for _, id := range e.prizeIDs {
		samplerWeights = append(samplerWeights, dynamicWeights[id])
		samplerIDs = append(samplerIDs, id)
	}

	// Build alias table on the fly for dynamic weights
	alias := NewAliasTable(samplerWeights)
	if alias == nil {
		return result
	}

	idx := alias.Next()
	if idx < 0 || idx >= len(samplerIDs) {
		return result
	}

	selectedID := samplerIDs[idx]
	if selectedID == "" {
		// Miss — increment pity counter
		tracker.IncrementMiss(userID, campaignID)
		return result
	}

	// Won a prize
	result.PrizeID = selectedID
	result.PrizeLevel = e.prizeLevels[selectedID]
	tracker.Reset(userID, campaignID)
	return result
}

// DrawMultiple performs N draws sequentially. Useful for 10-pull scenarios.
func (e *Engine) DrawMultiple(n int, cfg PityConfig, tracker PityTracker, userID, campaignID string) []DrawResult {
	results := make([]DrawResult, n)
	for i := 0; i < n; i++ {
		results[i] = e.Draw(cfg, tracker, userID, campaignID)
	}
	return results
}

// MonteCarloResult holds the output of a multi-simulation run.
type MonteCarloResult struct {
	TotalDraws        int
	Wins              int
	Misses            int
	WinRate           float64
	PrizeWins         map[string]int
	PrizeWinRates     map[string]float64
	HardPityTriggers int
	AvgMissesToWin    float64
	TotalMisses       int
}

// Simulate runs a Monte Carlo simulation of the drawing process.
// numSimulations: how many full "from 0 to first win" cycles to run.
func (e *Engine) Simulate(numSimulations int, cfg PityConfig) MonteCarloResult {
	// Use a fresh in-memory tracker for simulation
	tracker := NewMemoryPityTracker()
	result := MonteCarloResult{
		PrizeWins:     make(map[string]int),
		PrizeWinRates: make(map[string]float64),
	}

	for i := 0; i < numSimulations; i++ {
		userID := fmt.Sprintf("sim_user_%d", i)
		campaignID := "sim_campaign"

		for {
			draw := e.Draw(cfg, tracker, userID, campaignID)
			result.TotalDraws++

			if draw.PrizeID != "" {
				result.Wins++
				result.PrizeWins[draw.PrizeID]++
				if draw.IsHardPity {
					result.HardPityTriggers++
				}
				break
			} else {
				result.Misses++
			}
		}

		state := tracker.Get(userID, campaignID)
		result.TotalMisses += state.ConsecutiveMisses
	}

	if result.Wins > 0 {
		result.WinRate = float64(result.Wins) / float64(result.TotalDraws) * 100
		result.AvgMissesToWin = float64(result.TotalMisses) / float64(result.Wins)
	}

	totalWins := result.Wins
	if totalWins > 0 {
		for id, count := range result.PrizeWins {
			result.PrizeWinRates[id] = float64(count) / float64(totalWins) * 100
		}
	}

	return result
}

// String returns a human-readable summary of the simulation.
func (r MonteCarloResult) String() string {
	s := fmt.Sprintf("=== Monte Carlo Simulation ===\n")
	s += fmt.Sprintf("Total Draws:    %d\n", r.TotalDraws)
	s += fmt.Sprintf("Wins:           %d (%.2f%%)\n", r.Wins, r.WinRate)
	s += fmt.Sprintf("Misses:         %d\n", r.Misses)
	s += fmt.Sprintf("Hard Pity Hits: %d\n", r.HardPityTriggers)
	s += fmt.Sprintf("Avg Misses/Win: %.2f\n", r.AvgMissesToWin)
	s += fmt.Sprintf("--- Per-Prize Breakdown ---\n")
	// Sort for deterministic output
	ids := make([]string, 0, len(r.PrizeWins))
	for id := range r.PrizeWins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		s += fmt.Sprintf("  %s: %d (%.2f%% of wins)\n", id, r.PrizeWins[id], r.PrizeWinRates[id])
	}
	return s
}
