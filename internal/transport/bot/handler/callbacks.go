package handler

import (
	"fmt"
	"strings"
	"tg_market/internal/domain/entity"
	"tg_market/internal/transport/bot/view"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

func (h *Handler) OnCatalogCallback(ctx *th.Context, query telego.CallbackQuery) error {
	// 1. Парсим номер страницы. Формат: "catalog_page:<number>"
	var page int
	// Используем Sscanf с двоеточием, чтобы соответствовать генератору клавиатуры
	_, err := fmt.Sscanf(query.Data, "catalog_page:%d", &page)
	if err != nil || page < 1 {
		page = 1
	}

	// 2. Получаем ВСЕ подарки сразу (так как их всего ~100)
	// Лимит 1000 с запасом.
	allGifts, err := h.svc.ListGiftTypes(ctx, 1000, 0)
	if err != nil {
		// Сообщаем об ошибке всплывающим уведомлением (Alert)
		_ = ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(query.ID).
			WithText("❌ Ошибка получения данных").WithShowAlert())
		return err
	}

	totalCount := len(allGifts)
	limit := 10
	totalPages := (totalCount + limit - 1) / limit

	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	start := (page - 1) * limit
	end := start + limit
	if end > totalCount {
		end = totalCount
	}

	var pageGifts []entity.GiftType
	if start < totalCount {
		pageGifts = allGifts[start:end]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 <b>Каталог подарков</b> (Стр. %d/%d)\n\n", page, totalPages))

	for _, gift := range pageGifts {
		// Если используете view.CatalogItemTemplate, то fmt.Sprintf(view..., ...)
		sb.WriteString(fmt.Sprintf(view.CatalogItemTemplate, gift.Name, gift.ID, gift.AveragePrice))
	}

	keyboard := createPaginationKeyboard(page, totalPages)

	_, err = ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:      tu.ID(query.Message.GetChat().ID),
		MessageID:   query.Message.GetMessageID(),
		Text:        sb.String(),
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: keyboard,
	})

	// Если сообщение не изменилось (пользователь нажал на ту же страницу), Telegram вернет ошибку.
	// Обычно её игнорируют, но можно залогировать.
	if err != nil {
		// Log error if needed
	}

	// 7. Обязательно отвечаем на коллбэк, чтобы убрать часики
	_ = ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(query.ID))

	return nil
}
