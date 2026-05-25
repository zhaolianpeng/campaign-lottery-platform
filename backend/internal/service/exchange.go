package service

import (
	"fmt"

	"campaign-lottery-platform/backend/internal/model"
)

// ============================================================
// 交换市场
// ============================================================

func (s *Service) ExchangeOffers() []model.ExchangeOffer {
	return s.store.ExchangeOffers()
}

func (s *Service) CreateExchangeOffer(token string, input model.ExchangeOfferMutation) (model.ExchangeOffer, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	// 验证用户是否拥有 HavePrize
	hasIt, err := s.store.UserHasPrize(user.ID, input.HavePrizeID)
	if err != nil || !hasIt {
		return model.ExchangeOffer{}, fmt.Errorf("you don't own this prize")
	}
	return s.store.CreateExchangeOffer(user.ID, input)
}

func (s *Service) CancelExchangeOffer(token string, offerID string) error {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return err
	}
	return s.store.CancelExchangeOffer(user.ID, offerID)
}

func (s *Service) AcceptExchangeOffer(token string, offerID string) (model.ExchangeOffer, error) {
	user, err := s.store.UserFromToken(token)
	if err != nil {
		return model.ExchangeOffer{}, err
	}
	return s.store.AcceptExchangeOffer(user.ID, offerID)
}
