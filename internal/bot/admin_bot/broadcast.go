package admin_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) executeBroadcast(ctx context.Context, chatID int64, userID int64, messageText string, months int) {
	inactiveSince := time.Now().AddDate(0, -months, 0)
	clients, err := h.repos.Client.GetAllForBroadcast(ctx, h.inst.Master.ID, inactiveSince)
	if err != nil || len(clients) == 0 {
		h.inst.SendMessage(chatID, "Клиентов для рассылки не найдено.")
		return
	}

	h.inst.SendMessage(chatID, fmt.Sprintf("📤 Отправляю рассылку %d клиентам...", len(clients)))

	sent := 0
	failed := 0

	for _, client := range clients {
		// Personalize message — replace [Имя] with client name
		personalText := strings.ReplaceAll(messageText, "[Имя]", client.Name)
		if client.Name == "" {
			personalText = strings.ReplaceAll(personalText, "[Имя],", "")
			personalText = strings.TrimSpace(personalText)
		}

		// Add unsubscribe button
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 Записаться", "booking_start"),
				tgbotapi.NewInlineKeyboardButtonData("🔕 Отписаться", "no_broadcast"),
			),
		)

		// Check quiet hours
		hour := time.Now().Hour()
		if hour < 5 || hour >= 21 {
			failed++
			continue
		}

		msg := tgbotapi.NewMessage(client.TelegramID, personalText)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard

		// Get client bot for this master
		err := h.inst.Notifier.SendToClient(h.inst.Master.ID, client.TelegramID, personalText, &keyboard)
		if err != nil {
			failed++
			// Mark as blocked if error contains "blocked"
			if strings.Contains(err.Error(), "blocked") {
				h.repos.Client.MarkBlocked(ctx, h.inst.Master.ID, client.TelegramID, true)
			}
		} else {
			sent++
		}

		// Small delay to avoid Telegram rate limits
		time.Sleep(50 * time.Millisecond)
	}

	// Send stats to master
	text := fmt.Sprintf(
		"Рассылка отправлена ✅\n\n"+
			"📤 Отправлено: %d\n"+
			"✅ Доставлено: %d\n"+
			"❌ Не доставлено: %d",
		len(clients), sent, failed,
	)
	h.inst.SendMessage(chatID, text)
}
