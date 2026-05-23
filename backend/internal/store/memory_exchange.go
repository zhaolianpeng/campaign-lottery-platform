package store

import (
	"errors"
	"time"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 交换市场
// ============================================================

func (s *MemoryStore) ExchangeOffers() []model.ExchangeOffer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.ExchangeOffer, len(s.exchangeOffers))
	copy(items, s.exchangeOffers)
	return items
}

func (s *MemoryStore) CreateExchangeOffer(userID string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offer := model.ExchangeOffer{
		ID:          "ex_offer_" + randomSuffix(10),
		UserID:      userID,
		HavePrizeID: input.HavePrizeID,
		WantPrizeID: input.WantPrizeID,
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}
	if u, ok := s.users[userID]; ok {
		offer.UserNickname = u.Nickname
	}
	// 填充名称
	for _, inv := range s.inventory {
		if inv.PrizeID == input.HavePrizeID {
			offer.HavePrizeName = inv.PrizeName
		}
	}
	for _, prizes := range s.prizes {
		for _, p := range prizes {
			if p.ID == input.WantPrizeID {
				offer.WantPrizeName = p.Name
			}
		}
	}

	s.exchangeOffers = append([]model.ExchangeOffer{offer}, s.exchangeOffers...)
	return offer, nil
}

func (s *MemoryStore) CancelExchangeOffer(userID, offerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.exchangeOffers {
		if s.exchangeOffers[i].ID == offerID {
			if s.exchangeOffers[i].UserID != userID {
				return errors.New("not your offer")
			}
			s.exchangeOffers[i].Status = "cancelled"
			return nil
		}
	}
	return errors.New("offer not found")
}

func (s *MemoryStore) AcceptExchangeOffer(userID, offerID string) (model.ExchangeOffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.exchangeOffers {
		if s.exchangeOffers[i].ID == offerID {
			if s.exchangeOffers[i].UserID == userID {
				return model.ExchangeOffer{}, errors.New("cannot accept your own offer")
			}
			if s.exchangeOffers[i].Status != "pending" {
				return model.ExchangeOffer{}, errors.New("offer is no longer pending")
			}
			s.exchangeOffers[i].Status = "matched"
			return s.exchangeOffers[i], nil
		}
	}
	return model.ExchangeOffer{}, errors.New("offer not found")
}
