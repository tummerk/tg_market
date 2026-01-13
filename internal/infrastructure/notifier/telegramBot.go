package notifier

import (
	"context"
	"fmt"
	"os/exec" // <--- 1. Добавили для запуска команд
	"runtime" // <--- 1. Добавили для определения ОС
	"tg_market/internal/domain/entity"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type TelegramBot struct {
	bot    *telego.Bot
	chatID int64
}

func NewTelegramBot(token string, chatID int64) (*TelegramBot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	return &TelegramBot{
		bot:    bot,
		chatID: chatID,
	}, nil
}

// Run запускает обработку сделок из канала.
func (b *TelegramBot) Run(ctx context.Context, deals <-chan entity.Deal) error {
	// Внешний цикл: читаем сделки из канала
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case deal, ok := <-deals:
			if !ok {
				return nil // Канал закрыт
			}

			// Внутренний цикл: "Retry forever"
			// Мы не выйдем отсюда, пока не отправим сделку или не умрет контекст
			for {
				err := b.SendDeal(ctx, deal)
				if err == nil {
					// Успех! Выходим из внутреннего цикла (break),
					// чтобы внешний цикл мог взять следующую сделку.
					break
				}

				// Логируем ошибку
				fmt.Printf("failed to send deal (retrying in 3s): %v\n", err)

				// Пауза перед повторной попыткой.
				// Используем select, чтобы не блокировать остановку программы.
				select {
				case <-ctx.Done():
					return ctx.Err() // Программу остановили во время ожидания
				case <-time.After(3 * time.Second):
					// Прошло 3 секунды, идем на следующий круг внутреннего for
				}
			}
		}
	}
}

func (b *TelegramBot) SendDeal(ctx context.Context, deal entity.Deal) error {
	// --- 3. ВЫЗЫВАЕМ ЗВУК ЗДЕСЬ ---
	// Запускаем в горутине, чтобы звук не тормозил отправку сообщения
	go playSound()
	// ------------------------------

	text := fmt.Sprintf(
		"🔥 <b>GEM FOUND!</b>\n\n"+
			"🎁 <b>Name:</b> %s\n"+
			"💰 <b>StarPrice:</b> %d ⭐\n"+
			"💰 <b>TonPrice:</b> %.2f\n"+
			"📊 <b>Avg StarPrice:</b> %d ⭐\n"+
			"📉 <b>Profit:</b> %.1f%%\n\n"+
			"🔗 <a href=\"%s\">Buy Now</a>",
		deal.GiftType.Name,
		deal.Gift.StarPrice,
		deal.Gift.TonPrice,
		deal.AvgPrice,
		deal.Profit,
		deal.Gift.Address,
	)
	fmt.Println(text)

	msg := tu.Message(
		tu.ID(b.chatID),
		text,
	).WithParseMode(telego.ModeHTML)

	_, err := b.bot.SendMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (b *TelegramBot) SendText(ctx context.Context, text string) error {
	msg := tu.Message(tu.ID(b.chatID), text)
	_, err := b.bot.SendMessage(ctx, msg)
	return err
}

// --- 2. ФУНКЦИЯ ВОСПРОИЗВЕДЕНИЯ ЗВУКА ---
func playSound() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("afplay", "/System/Library/Sounds/Glass.aiff")
	case "windows":
		cmd = exec.Command("powershell", "-c", "[System.Console]::Beep(1000, 500)")
	default:
		return
	}

	if cmd != nil {
		_ = cmd.Run()
	}
}
