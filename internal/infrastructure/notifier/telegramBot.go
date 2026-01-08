package notifier

import (
	"context"
	"fmt"
	"tg_market/internal/domain/entity"

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

func (b *TelegramBot) SendDeal(ctx context.Context, deal entity.Deal) error {
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

// SendText отправляет простое текстовое сообщение.
func (b *TelegramBot) SendText(ctx context.Context, text string) error {
	msg := tu.Message(tu.ID(b.chatID), text)

	_, err := b.bot.SendMessage(ctx, msg)
	return err
}
