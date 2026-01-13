package handler

import (
	service "tg_market/internal/domain/service/gift"
	"tg_market/internal/worker"
)

type Handler struct {
	svc     *service.GiftService
	scanner *worker.MarketScanner
}

func New(svc *service.GiftService, scanner *worker.MarketScanner) *Handler {
	return &Handler{
		svc:     svc,
		scanner: scanner,
	}
}
