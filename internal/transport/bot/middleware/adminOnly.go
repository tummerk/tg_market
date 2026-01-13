package middleware

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func AdminOnly(adminID int64) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		var userID int64

		if update.Message != nil {
			userID = update.Message.From.ID
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
		} else {
			return nil
		}

		// ПРОВЕРКА
		if userID == adminID {
			return ctx.Next(update)
		}

		return nil
	}
}
