package admin_bot

import (
	"beauty-bot/internal/models"
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

func (h *Handler) handleBlockSlot(ctx context.Context, chatID int64) {
	now := time.Now()
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i := 0; i < 30; i++ {
		date := now.AddDate(0, 0, i)
		label := fmt.Sprintf("%d %s", date.Day(), shortMonth(date.Month()))
		cbData := fmt.Sprintf("block_slot_day_%s", date.Format("2006-01-02"))
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
	h.inst.SendWithInlineKeyboard(chatID, "Выберите день:", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleBlockSlotDay(ctx context.Context, chatID int64, dateStr string) {
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return
	}

	master := h.inst.Master
	s := master.Schedule

	// Определяем рабочие часы для этого дня
	weekday := date.Weekday()
	daySchedules := map[time.Weekday]models.DaySchedule{
		time.Monday:    s.Mon,
		time.Tuesday:   s.Tue,
		time.Wednesday: s.Wed,
		time.Thursday:  s.Thu,
		time.Friday:    s.Fri,
		time.Saturday:  s.Sat,
		time.Sunday:    s.Sun,
	}
	daySchedule := daySchedules[weekday]
	if daySchedule.Start == nil || daySchedule.End == nil {
		h.inst.SendMessage(chatID, "Этот день выходной, нечего блокировать.")
		return
	}

	// Генерируем слоты
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	start := time.Date(date.Year(), date.Month(), date.Day(),
		daySchedule.Start.Hour(), daySchedule.Start.Minute(), 0, 0, time.Local)
	end := time.Date(date.Year(), date.Month(), date.Day(),
		daySchedule.End.Hour(), daySchedule.End.Minute(), 0, 0, time.Local)

	for t := start; t.Before(end); t = t.Add(time.Duration(master.SlotIntervalMin) * time.Minute) {
		label := t.Format("15:04")
		cbData := fmt.Sprintf("block_slot_confirm_%s_%s", dateStr, t.Format("15:04"))
		btn := tgbotapi.NewInlineKeyboardButtonData(label, cbData)
		row = append(row, btn)
		if len(row) == 4 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "block_slot"),
	))
	h.inst.SendWithInlineKeyboard(chatID, fmt.Sprintf("Выберите слот на %d %s:", date.Day(), fullMonth(date.Month())),
		tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleBlockSlotConfirm(ctx context.Context, chatID int64, dateStr, timeStr string) {
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return
	}
	t, err := time.ParseInLocation("15:04", timeStr, time.Local)
	if err != nil {
		return
	}

	master := h.inst.Master
	slotStart := time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
	slotEnd := slotStart.Add(time.Duration(master.SlotIntervalMin) * time.Minute)

	err = h.repos.BlockedSlot.Create(ctx, master.ID, slotStart, slotEnd, "")
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка блокировки.")
		return
	}
	h.inst.SendMessage(chatID, fmt.Sprintf("Слот %s %d %s заблокирован ✅", timeStr, date.Day(), fullMonth(date.Month())))
}

func (h *Handler) handleBlockPeriod(ctx context.Context, chatID int64) {
	now := time.Now()
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i := 0; i < 60; i++ {
		date := now.AddDate(0, 0, i)
		label := fmt.Sprintf("%d %s", date.Day(), shortMonth(date.Month()))
		cbData := fmt.Sprintf("block_period_start_%s", date.Format("2006-01-02"))
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
	h.inst.SendWithInlineKeyboard(chatID, "Выберите дату начала отпуска:", tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleBlockPeriodEnd(ctx context.Context, chatID int64, startStr string) {
	start, err := time.ParseInLocation("2006-01-02", startStr, time.Local)
	if err != nil {
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i := 1; i <= 60; i++ {
		date := start.AddDate(0, 0, i)
		label := fmt.Sprintf("%d %s", date.Day(), shortMonth(date.Month()))
		cbData := fmt.Sprintf("block_period_confirm_%s_%s", startStr, date.Format("2006-01-02"))
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
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "block_period"),
	))
	h.inst.SendWithInlineKeyboard(chatID,
		fmt.Sprintf("Начало: %d %s\nВыберите дату окончания:", start.Day(), fullMonth(start.Month())),
		tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleBlockPeriodConfirm(ctx context.Context, chatID int64, startStr, endStr string) {
	start, err := time.ParseInLocation("2006-01-02", startStr, time.Local)
	if err != nil {
		return
	}
	end, err := time.ParseInLocation("2006-01-02", endStr, time.Local)
	if err != nil {
		return
	}

	// Проверяем есть ли записи в этот период
	bookings, _ := h.repos.Booking.GetForPeriod(ctx, h.inst.Master.ID, start, end.AddDate(0, 0, 1))
	if len(bookings) > 0 {
		h.inst.SendMessage(chatID, fmt.Sprintf(
			"В этот период есть %d записей. Сначала отмените их.", len(bookings)))
		return
	}

	periodStart := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	periodEnd := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)

	err = h.repos.BlockedSlot.Create(ctx, h.inst.Master.ID, periodStart, periodEnd, "отпуск")
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка блокировки.")
		return
	}

	h.inst.SendMessage(chatID, fmt.Sprintf(
		"Период заблокирован ✅\n%d %s — %d %s\nКлиенты не смогут записаться в эти дни.",
		start.Day(), fullMonth(start.Month()),
		end.Day(), fullMonth(end.Month()),
	))
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
