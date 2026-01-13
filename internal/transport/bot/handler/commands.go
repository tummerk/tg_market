package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tg_market/internal/transport/bot/view"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

func (h *Handler) OnStart(ctx *th.Context, msg telego.Message) error {
	_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: msg.Chat.ID},
		Text:      view.StartMessage,
		ParseMode: telego.ModeHTML,
	})
	return err
}

func (h *Handler) OnStatus(ctx *th.Context, msg telego.Message) error {
	scannerStatus := "🔴 остановлен"
	if h.scanner.IsRunning() {
		scannerStatus = "🟢 работает"
	}

	// Получаем список сканируемых ID
	scanList := "все товары из каталога"
	ids := h.scanner.GetGiftTypes()
	if len(ids) > 0 {
		scanList = fmt.Sprintf("%d выбранных товаров", len(ids))
	}

	text := fmt.Sprintf(`📊 <b>Статус системы</b>

	🔍 <b>Сканер:</b> %s
	📦 <b>Сканируется:</b> %s
	💰 <b>Лимит баланса:</b> %.2f ⭐
	📉 <b>Мин. скидка:</b> %.1f%%
	🛒 <b>Автопокупка:</b> %s
`,
		scannerStatus,
		scanList,
		h.svc.GetBalance(),
		h.svc.GetDiscount(),
		boolToStatus(h.svc.IsAutoBuyEnabled()),
	)

	return h.sendHTML(ctx, msg.Chat.ID, text)
}

func boolToStatus(b bool) string {
	if b {
		return "✅ вкл"
	}
	return "❌ выкл"
}

func (h *Handler) OnAutoBuy(ctx *th.Context, msg telego.Message) error {
	enabled := h.svc.SetAutoBuy()
	_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: msg.Chat.ID},
		Text:   boolToStatus(enabled),
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) OnSetBalance(ctx *th.Context, msg telego.Message) error {
	// Извлекаем текст сообщения после команды
	text := msg.Text

	// Разбиваем текст сообщения на части по пробелам
	parts := strings.Fields(text)
	// Проверяем, что есть команда и аргумент
	if len(parts) < 2 {
		_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.SetBalanceMissingArgument,
		})
		return err
	}

	// Берем второй элемент (первый после команды) как аргумент
	arg := parts[1]

	var amount float64
	_, err := fmt.Sscanf(arg, "%f", &amount)
	if err != nil || amount < 0 {
		_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.SetBalanceInvalidFormat,
		})
		return err
	}

	h.svc.SetBalance(amount)

	_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: msg.Chat.ID},
		Text:   fmt.Sprintf(view.SetBalanceSuccess, amount),
	})
	return err
}

func (h *Handler) OnSetDiscount(ctx *th.Context, msg telego.Message) error {
	// Извлекаем текст сообщения
	text := msg.Text

	// Разбиваем текст сообщения на части по пробелам
	parts := strings.Fields(text)

	// Проверяем, что есть команда и аргумент
	if len(parts) < 2 {
		_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.SetDiscountMissingArgument,
		})
		return err
	}

	// Берем второй элемент (первый после команды) как аргумент
	arg := parts[1]

	var percent float64
	_, err := fmt.Sscanf(arg, "%f", &percent)
	if err != nil || percent < 0 || percent > 100 {
		_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.SetDiscountInvalidFormat,
		})
		return err
	}

	h.svc.SetDiscount(percent)

	_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: msg.Chat.ID},
		Text:   fmt.Sprintf(view.SetDiscountSuccess, percent),
	})
	return err
}

func (h *Handler) OnStartScan(ctx *th.Context, msg telego.Message) error {
	// Проверяем, запущен ли уже сканер
	if h.scanner.IsRunning() {
		_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   "Сканер уже запущен!",
		})
		return err
	}

	// Запускаем сканер
	err := h.scanner.Start(context.Background())
	if err != nil {
		_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   fmt.Sprintf("Ошибка запуска сканера: %v", err),
		})
		return err
	}

	_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: msg.Chat.ID},
		Text:   "Сканер запущен!",
	})
	return err
}

func (h *Handler) OnStopScan(ctx *th.Context, msg telego.Message) error {
	// Проверяем, запущен ли сканер
	if !h.scanner.IsRunning() {
		_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   "Сканер не запущен!",
		})
		return err
	}

	// Останавливаем сканер
	h.scanner.Stop()

	_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: msg.Chat.ID},
		Text:   "Сканер остановлен!",
	})
	return err
}

func (h *Handler) OnCatalog(ctx *th.Context, msg telego.Message) error {
	page := 1
	limit := 10
	offset := (page - 1) * limit

	// Получаем общее количество подарков для вычисления количества страниц
	totalGiftTypes, err := h.svc.ListGiftTypes(ctx, 100, 0) // получаем все для подсчета общего количества
	if err != nil {
		_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.CatalogError,
		})
		return err
	}

	totalCount := len(totalGiftTypes)
	totalPages := (totalCount + limit - 1) / limit // округление вверх

	// Получаем список типов подарков для текущей страницы
	giftTypes, err := h.svc.ListGiftTypes(ctx, limit, offset)
	if err != nil {
		_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.CatalogError,
		})
		return err
	}

	if len(giftTypes) == 0 {
		_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: msg.Chat.ID},
			Text:   view.CatalogEmpty,
		})
		return err
	}

	// Формируем сообщение с каталогом
	catalogText := fmt.Sprintf(view.CatalogPaginationTemplate, page, totalPages)

	for _, giftType := range giftTypes {
		catalogText += fmt.Sprintf(
			view.CatalogItemTemplate,
			giftType.Name,
			giftType.ID,
			giftType.AveragePrice,
		)
	}

	// Создаем инлайн-клавиатуру для пагинации
	inlineKeyboard := createPaginationKeyboard(page, totalPages)

	_, err = ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: msg.Chat.ID},
		Text:        catalogText,
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: inlineKeyboard,
	})
	return err
}

func createPaginationKeyboard(page, totalPages int) *telego.InlineKeyboardMarkup {
	var buttons []telego.InlineKeyboardButton

	if page > 1 {
		buttons = append(buttons, tu.InlineKeyboardButton("⬅️").
			WithCallbackData(fmt.Sprintf("catalog_page:%d", page-1)))
	}

	buttons = append(buttons, tu.InlineKeyboardButton(fmt.Sprintf("%d / %d", page, totalPages)).
		WithCallbackData("noop")) // noop = no operation

	if page < totalPages {
		buttons = append(buttons, tu.InlineKeyboardButton("➡️").
			WithCallbackData(fmt.Sprintf("catalog_page:%d", page+1)))
	}

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(buttons...),
	)
}

func (h *Handler) OnSync(ctx *th.Context, msg telego.Message) error {
	return nil
}

func (h *Handler) OnUpdatePrices(ctx *th.Context, msg telego.Message) error {
	return nil
}

func (h *Handler) OnScanGems(ctx *th.Context, msg telego.Message) error {
	return nil
}

func (h *Handler) OnAddScan(ctx *th.Context, msg telego.Message) error {
	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		return h.sendHTML(ctx, msg.Chat.ID, "❌ Использование: /addscan <code>ID</code>")
	}

	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return h.sendHTML(ctx, msg.Chat.ID, "❌ Неверный формат ID")
	}

	// Проверяем, есть ли уже
	if h.scanner.HasGiftType(id) {
		return h.sendHTML(ctx, msg.Chat.ID, fmt.Sprintf("⚠️ ID <code>%d</code> уже в списке", id))
	}

	h.scanner.AddGiftType(id)

	return h.sendHTML(ctx, msg.Chat.ID,
		fmt.Sprintf("✅ ID <code>%d</code> добавлен\n📊", id))
}

// OnRemoveScan удаляет товар из сканирования
// Использование: /removescan 5882260270843168924
func (h *Handler) OnRemoveScan(ctx *th.Context, msg telego.Message) error {
	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		return h.sendHTML(ctx, msg.Chat.ID, "❌ Использование: /removescan <code>ID</code>")
	}

	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return h.sendHTML(ctx, msg.Chat.ID, "❌ Неверный формат ID")
	}

	if !h.scanner.HasGiftType(id) {
		return h.sendHTML(ctx, msg.Chat.ID, fmt.Sprintf("⚠️ ID <code>%d</code> не найден в списке", id))
	}

	h.scanner.RemoveGiftType(id)

	text := fmt.Sprintf("✅ ID <code>%d</code> удалён\n📊", id)

	return h.sendHTML(ctx, msg.Chat.ID, text)
}

// OnListScan показывает текущий список сканируемых товаров
func (h *Handler) OnListScan(ctx *th.Context, msg telego.Message) error {
	ids := h.scanner.GetGiftTypes()

	if len(ids) == 0 {
		text := "📋 <b>Список сканирования пуст</b>\n\n" +
			"Сканируются все товары из каталога.\n\n" +
			"Добавить товар: /addscan <code>ID</code>"
		return h.sendHTML(ctx, msg.Chat.ID, text)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Сканируемые товары (%d):</b>\n\n", len(ids)))

	for i, id := range ids {
		// Пытаемся получить название товара
		name := "неизвестный"
		giftType, err := h.svc.GetGiftType(ctx, id)
		if err == nil && giftType != nil {
			name = giftType.Name
		}

		sb.WriteString(fmt.Sprintf("%d. <code>%d</code> (%s)\n", i+1, id, name))
	}

	sb.WriteString("\n<i>Нажмите на ID чтобы скопировать</i>")

	return h.sendHTML(ctx, msg.Chat.ID, sb.String())
}

// OnClearScan очищает список — будут сканироваться все товары
func (h *Handler) OnClearScan(ctx *th.Context, msg telego.Message) error {
	h.scanner.ClearGiftTypes()

	return h.sendHTML(ctx, msg.Chat.ID,
		fmt.Sprintf("✅ Список очищен \n\n💡 Теперь сканируются все товары из каталога"))
}

// OnSetScan устанавливает список ID (заменяет текущий)
// Использование: /setscan 123 456 789
func (h *Handler) OnSetScan(ctx *th.Context, msg telego.Message) error {
	args := strings.Fields(msg.Text)

	if len(args) < 2 {
		return h.sendHTML(ctx, msg.Chat.ID,
			"❌ Использование: /setscan <code>ID1</code> <code>ID2</code> ...\n\n"+
				"Пример: /setscan 123456 789012 345678")
	}

	var ids []int64
	var errors []string

	for _, arg := range args[1:] {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			errors = append(errors, arg)
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return h.sendHTML(ctx, msg.Chat.ID, "❌ Не удалось распознать ни одного ID")
	}

	h.scanner.SetGiftTypes(ids)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Установлено %d товаров для сканирования:\n\n", len(ids)))

	for i, id := range ids {
		sb.WriteString(fmt.Sprintf("%d. <code>%d</code>\n", i+1, id))
	}

	if len(errors) > 0 {
		sb.WriteString(fmt.Sprintf("\n⚠️ Пропущены неверные ID: %s", strings.Join(errors, ", ")))
	}

	return h.sendHTML(ctx, msg.Chat.ID, sb.String())
}

// Вспомогательные методы

func (h *Handler) sendHTML(ctx *th.Context, chatID int64, text string) error {
	_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: "HTML",
	})
	return err
}

func (h *Handler) send(ctx *th.Context, chatID int64, text string) error {
	_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
	})
	return err
}
