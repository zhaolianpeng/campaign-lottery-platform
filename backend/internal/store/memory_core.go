package store

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

type MemoryStore struct {
	mu             sync.RWMutex
	users          map[string]model.User
	sessions       map[string]model.Session
	adminSessions  map[string]time.Time
	campaigns      map[string]model.Campaign
	prizes         map[string][]model.Prize
	drawRecords    []model.DrawRecord
	userDrawCounts map[string]int
	adminUser      string
	adminPassword  string

	// 盲盒扩展
	inventory       []model.UserInventory
	exchangeOffers  []model.ExchangeOffer
	members         map[string]model.UserMember
	pointsLog       []model.UserPointsLog
	nextTaskID      int64

	// 集卡系统扩展
	checkInDates   map[string]time.Time
	checkInStreaks map[string]int
	shareCounts    map[string]int

	// 月卡系统
	monthCards   map[string]*model.MonthCard
	freeDrawUsed map[string]int

	// 月卡/付费卡系统
	userCards map[string]model.UserCard

	// 战令系统
	battlePasses          map[string]*model.BattlePass
	battlePassSeasons     map[int]model.BattlePassSeason
	battlePassTasks       []model.BattlePassTask
	battlePassTaskProgress map[string]model.BattlePassTaskProgress
	battlePassRewards     []model.BattlePassReward
	nextSeasonID          int
	nextTaskIDCounter     int

	// 商店 + 道具 + 首充
	shopItems            []model.ShopItem
	userItems            map[string]model.UserItem
	firstRechargeRecords map[string]model.UserFirstRecharge

	// v1.5 社交裂变
	inviteRecords       []model.InviteRecord
	assistProgress      map[string]model.AssistProgress
	assistActions       []model.AssistAction
	teamIDSeq           int
	teams               map[string]model.Team
	teamMembers         map[string][]model.TeamMember
	giftRecords         []model.GiftRecord
	giftIDSeq           int
	shareCards          map[string][]model.ShareCard
	inviteIDSeq         int
	assistIDSeq         int
	shareCardIDSeq      int
	teamRewards         map[string]model.TeamReward
	giftPrizeInfo       map[string]string

	// v1.6 碎片拼图
	puzzleTemplates  []model.PuzzleTemplate
	puzzleProgresses map[string]*model.PuzzleProgress
	puzzleTeams      map[string]*model.PuzzleTeam
	puzzleTeamIDSeq  int

	// v1.6 预约抢购
	flashSales         []model.FlashSale
	flashSubscriptions []model.FlashSubscription
	flashIDSeq         int

	// 活动系统
	activities              []model.Activity
	activityParticipations  []model.ActivityParticipation
	activityRewards         []model.ActivityReward
	activityIDSeq           int
}

func NewMemoryStore(adminUser string, adminPassword string) *MemoryStore {
	store := &MemoryStore{
		users:          make(map[string]model.User),
		sessions:       make(map[string]model.Session),
		adminSessions:  make(map[string]time.Time),
		campaigns:      make(map[string]model.Campaign),
		prizes:         make(map[string][]model.Prize),
		drawRecords:    make([]model.DrawRecord, 0, 16),
		userDrawCounts: make(map[string]int),
		adminUser:      adminUser,
		adminPassword:  adminPassword,
		inventory:      make([]model.UserInventory, 0, 32),
		exchangeOffers: make([]model.ExchangeOffer, 0, 8),
		members:        make(map[string]model.UserMember),
		pointsLog:      make([]model.UserPointsLog, 0, 16),
		nextTaskID:     1,
		checkInDates:   make(map[string]time.Time),
		checkInStreaks: make(map[string]int),
		shareCounts:    make(map[string]int),
		monthCards:     make(map[string]*model.MonthCard),
		freeDrawUsed:   make(map[string]int),
		userCards:      make(map[string]model.UserCard),
		battlePassSeasons:      make(map[int]model.BattlePassSeason),
		battlePasses:           make(map[string]*model.BattlePass),
		battlePassTasks:        make([]model.BattlePassTask, 0),
		battlePassTaskProgress: make(map[string]model.BattlePassTaskProgress),
		battlePassRewards:      make([]model.BattlePassReward, 0),
		shopItems:              make([]model.ShopItem, 0),
		userItems:              make(map[string]model.UserItem),
		firstRechargeRecords:   make(map[string]model.UserFirstRecharge),
		inviteRecords:          make([]model.InviteRecord, 0),
		assistProgress:         make(map[string]model.AssistProgress),
		assistActions:          make([]model.AssistAction, 0),
		teams:                  make(map[string]model.Team),
		teamMembers:            make(map[string][]model.TeamMember),
		giftRecords:            make([]model.GiftRecord, 0),
		shareCards:             make(map[string][]model.ShareCard),
		teamRewards:            make(map[string]model.TeamReward),
		giftPrizeInfo:          make(map[string]string),
		puzzleTemplates:        make([]model.PuzzleTemplate, 0),
		puzzleProgresses:       make(map[string]*model.PuzzleProgress),
		puzzleTeams:            make(map[string]*model.PuzzleTeam),
		puzzleTeamIDSeq:        0,
		flashSales:             make([]model.FlashSale, 0),
		flashSubscriptions:     make([]model.FlashSubscription, 0),
		flashIDSeq:             0,
		activities:             make([]model.Activity, 0),
		activityParticipations: make([]model.ActivityParticipation, 0),
		activityRewards:        make([]model.ActivityReward, 0),
		activityIDSeq:          0,
	}

	store.seedDefaultCampaign()
	store.seedBattlePass()
	store.seedShop()
	store.seedPuzzle()
	store.seedActivities()
	return store
}

func (s *MemoryStore) Seed() error { return nil }

// ============================================================
// 内部辅助
// ============================================================

func (s *MemoryStore) ensureAdmin(token string) error {
	expiresAt, ok := s.adminSessions[token]
	if !ok || expiresAt.Before(time.Now().UTC()) {
		return ErrAdminUnauthorized
	}
	return nil
}

func randomSuffix(size int) string {
	buffer := make([]byte, size)
	_, err := rand.Read(buffer)
	if err != nil {
		return time.Now().Format("150405.999999999")
	}
	return strings.ToLower(hex.EncodeToString(buffer))[:size]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ptrTime(t time.Time) *time.Time { return &t }

// GenerateID generates a random hex ID using crypto/rand (8 bytes → 16 hex chars)
func (s *MemoryStore) GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// getUserActiveTeamLocked is an internal helper that assumes the read lock is held.
func (s *MemoryStore) getUserActiveTeamLocked(userID string) *model.Team {
	for _, team := range s.teams {
		if team.Status != "recruiting" && team.Status != "active" {
			continue
		}
		members := s.teamMembers[team.ID]
		for _, m := range members {
			if m.UserID == userID {
				return &team
			}
		}
	}
	return nil
}
