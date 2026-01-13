package handler

import (
	"tg_market/internal/transport/bot/middleware"

	th "github.com/mymmrac/telego/telegohandler"
)

func (h *Handler) RegisterRoutes(bh *th.BotHandler, adminID int64) {
	// 1. Создаем группу, защищенную миддлварью
	// Используем ту миддлварь, которую мы писали в прошлом ответе
	adminGroup := bh.Group(th.AnyMessage())
	adminGroup.Use(middleware.AdminOnly(adminID))

	// 2. Регистрируем хендлеры ВНУТРИ группы
	// Команда /start
	adminGroup.HandleMessage(h.OnStart, th.CommandEqual("start"))

	// Команда /status
	adminGroup.HandleMessage(h.OnStatus, th.CommandEqual("status"))

	// Команда /autobuy
	adminGroup.HandleMessage(h.OnAutoBuy, th.CommandEqual("autobuy"))

	// Команда /setbalance
	adminGroup.HandleMessage(h.OnSetBalance, th.CommandEqual("setbalance"))

	// Команда /setdiscount
	adminGroup.HandleMessage(h.OnSetDiscount, th.CommandEqual("setdiscount"))

	// Команда /catalog
	adminGroup.HandleMessage(h.OnCatalog, th.CommandEqual("catalog"))

	// Команда /sync
	adminGroup.HandleMessage(h.OnSync, th.CommandEqual("sync"))

	// Команда /updateprices
	adminGroup.HandleMessage(h.OnUpdatePrices, th.CommandEqual("updateprices"))

	// Команда /scangems
	adminGroup.HandleMessage(h.OnScanGems, th.CommandEqual("scangems"))

	// Команда /startscan
	adminGroup.HandleMessage(h.OnStartScan, th.CommandEqual("startscan"))

	// Команда /stopscan
	adminGroup.HandleMessage(h.OnStopScan, th.CommandEqual("stopscan"))

	bh.HandleMessage(h.OnAddScan, th.CommandEqual("addscan"))
	bh.HandleMessage(h.OnRemoveScan, th.CommandEqual("removescan"))
	bh.HandleMessage(h.OnListScan, th.CommandEqual("listscan"))
	bh.HandleMessage(h.OnClearScan, th.CommandEqual("clearscan"))
	bh.HandleMessage(h.OnSetScan, th.CommandEqual("setscan"))

	// Обработчик callback-запросов для пагинации каталога
	adminGroup.HandleCallbackQuery(h.OnCatalogCallback, th.AnyCallbackQuery())

	cbGroup := bh.Group(th.AnyCallbackQuery())
	cbGroup.Use(middleware.AdminOnly(adminID))

	cbGroup.HandleCallbackQuery(h.OnCatalogCallback, th.CallbackDataPrefix("catalog_page"))
}
