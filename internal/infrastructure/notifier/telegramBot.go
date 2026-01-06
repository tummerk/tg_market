package notifier

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	service "tg_market/internal/domain/service/gift"
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
func (b *TelegramBot) Run(ctx context.Context, deals <-chan service.GoodDeal) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case deal, ok := <-deals:
			if !ok {
				return nil
			}
			if err := b.SendDeal(ctx, deal); err != nil {
				logger(ctx).Error("failed to send deal", "error", err)
			}
		}
	}
}

func (b *TelegramBot) SendDeal(ctx context.Context, deal service.GoodDeal) error {
	text := fmt.Sprintf(
		"🔥 <b>GEM FOUND!</b>\n\n"+
			"🎁 <b>Name:</b> %s\n"+
			"💰 <b>Price:</b> %d ⭐\n"+
			"📊 <b>Avg Price:</b> %d ⭐\n"+
			"📉 <b>Discount:</b> %.1f%%\n\n"+
			"🔗 <a href=\"%s\">Buy Now</a>",
		deal.GiftType.Name,
		deal.Gift.Price,
		deal.AvgPrice,
		deal.Discount,
		deal.Gift.Address,
	)

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

// SendText отправляет простое текстовое сообщение.
func (b *TelegramBot) SendText(ctx context.Context, text string) error {
	msg := tu.Message(tu.ID(b.chatID), text)

	_, err := b.bot.SendMessage(ctx, msg)
	return err
}
