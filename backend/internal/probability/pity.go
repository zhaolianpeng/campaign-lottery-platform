package probability

import (
	"math"
	"sync"
)

// PityState tracks the soft/hard pity state for a user in a campaign.
type PityState struct {
	UserID             string
	CampaignID         string
	ConsecutiveMisses  int     // 连续未中次数
	PityMultiplier     float64 // 当前概率倍数 (由 soft pity 计算得出)
}

// PityConfig defines the pity system parameters for a campaign/series.
type PityConfig struct {
	Enabled      bool    // 是否启用保底
	SoftPityN    int     // 软保底开始递增的次数
	PityFactor   float64 // 概率递增因子 α (每抽递增比例)
	HardPityN    int     // 硬保底次数 (必出)
	TargetWeight float64 // 保底目标奖品的基准权重
}

// PityTracker is the interface for tracking pity states across sessions.
type PityTracker interface {
	// Get returns the current pity state for a user + campaign.
	Get(userID, campaignID string) *PityState

	// IncrementMiss increments the consecutive miss counter and returns the updated state.
	IncrementMiss(userID, campaignID string) *PityState

	// Reset resets the pity state for a user + campaign (when they win).
	Reset(userID, campaignID string)
}

// MemoryPityTracker is an in-memory implementation of PityTracker.
type MemoryPityTracker struct {
	mu     sync.RWMutex
	states map[string]*PityState // key = "userID:campaignID"
}

// NewMemoryPityTracker creates a new in-memory pity tracker.
func NewMemoryPityTracker() *MemoryPityTracker {
	return &MemoryPityTracker{
		states: make(map[string]*PityState),
	}
}

func (t *MemoryPityTracker) Get(userID, campaignID string) *PityState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	state := t.states[userID+":"+campaignID]
	if state == nil {
		return &PityState{
			UserID:            userID,
			CampaignID:        campaignID,
			ConsecutiveMisses: 0,
			PityMultiplier:    1.0,
		}
	}
	// Return a copy
	s := *state
	return &s
}

func (t *MemoryPityTracker) IncrementMiss(userID, campaignID string) *PityState {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := userID + ":" + campaignID
	state, ok := t.states[key]
	if !ok {
		state = &PityState{
			UserID:     userID,
			CampaignID: campaignID,
		}
		t.states[key] = state
	}
	state.ConsecutiveMisses++
	return t.calcPityMultiplier(state)
}

func (t *MemoryPityTracker) Reset(userID, campaignID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.states, userID+":"+campaignID)
}

// calcPityMultiplier computes the effective probability multiplier based on
// the soft pity formula: P_actual(n) = P_base × (1 + α × n)
func (t *MemoryPityTracker) calcPityMultiplier(state *PityState) *PityState {
	n := state.ConsecutiveMisses
	state.PityMultiplier = 1.0 + (1.0 / float64(n+1)) // default fallback
	return state
}

// CalculateEffectiveProb computes the effective probability for a prize
// given the pity config and current consecutive misses.
// Returns the adjusted probability weight.
func CalculateEffectiveProb(baseWeight float64, cfg PityConfig, consecutiveMisses int) float64 {
	if !cfg.Enabled || cfg.SoftPityN <= 0 || consecutiveMisses < cfg.SoftPityN {
		return baseWeight
	}

	// Soft pity: probability increases with consecutive misses
	missesPastThreshold := consecutiveMisses - cfg.SoftPityN + 1
	multiplier := 1.0 + cfg.PityFactor*float64(missesPastThreshold)

	// Hard pity: cap at 100% (guaranteed win)
	if cfg.HardPityN > 0 && consecutiveMisses >= cfg.HardPityN {
		return math.MaxFloat64 // essentially guaranteed
	}

	// Apply multiplier but don't exceed the hard pity trigger
	effective := baseWeight * multiplier

	return effective
}
