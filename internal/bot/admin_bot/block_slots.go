package admin_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleBlockSlots(ctx context.Context, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Конкретный слот", "block_slot"),
			tgbotapi.NewInlineKeyboardButtonData("Весь день", "block_day"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Период (отпуск)", "block_period"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, "🚫 Заблокировать время:", keyboard)
}

func (h *Handler) handleBlockDay(ctx context.Context, chatID int64, userID int64) {
	// Show calendar for next 30 days
	now := time.Now()
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i := 0; i < 30; i++ {
		date := now.AddDate(0, 0, i)
		label := fmt.Sprintf("%d %s", date.Day(), shortMonth(date.Month()))
		cbData := fmt.Sprintf("block_day_confirm_%s", date.Format("2006-01-02"))
		btn := tgbotapi.NewInlineKeyboardButtonData(label, cbData)
		row = append(row, btn)
		if len(row) == 3 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "block_menu"),
	))

	h.inst.SendWithInlineKeyboard(chatID, "Выберите день для блокировки:",
		tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleBlockDayConfirm(ctx context.Context, chatID int64, dateStr string) {
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return
	}

	// Check if there are bookings on this day
	bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, date)
	if len(bookings) > 0 {
		var names []string
		for _, b := range bookings {
			names = append(names, fmt.Sprintf("%s — %s", b.StartsAt.Format("15:04"), b.ClientName))
		}
		text := fmt.Sprintf(
			"На этот день есть %d записей:\n%s\n\nСначала отмените их или заблокируйте только свободное время.",
			len(bookings),
			strings.Join(names, "\n"),
		)
		h.inst.SendMessage(chatID, text)
		return
	}

	// Block entire day
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	dayEnd := dayStart.Add(24 * time.Hour)

	_, err = h.inst.API.Request(tgbotapi.NewMessage(chatID, "")) // dummy — use db directly
	_ = err

	err = h.repos.BlockedSlot.Create(ctx, h.inst.Master.ID, dayStart, dayEnd, "")
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка блокировки.")
		return
	}

	h.inst.SendMessage(chatID,
		fmt.Sprintf("День заблокирован ✅\nКлиенты не смогут записаться на %d %s.",
			date.Day(), fullMonth(date.Month())))
}

func shortMonth(m time.Month) string {
	months := []string{"", "янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек"}
	return months[m]
}

func fullMonth(m time.Month) string {
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return months[m]
}
