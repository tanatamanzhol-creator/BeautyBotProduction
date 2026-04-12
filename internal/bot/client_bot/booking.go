package client_bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"beauty-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ── Step 1: Show services ──────────────────────────────────────────────────

func (h *Handler) handleBookingStart(ctx context.Context, msg *tgbotapi.Message, client *models.Client) {
	h.handleBookingStartCallback(ctx, msg.Chat.ID, client)
}

func (h *Handler) handleBookingStartCallback(ctx context.Context, chatID int64, client *models.Client) {
	services, err := h.repos.Service.GetActive(ctx, h.inst.Master.ID)
	if err != nil || len(services) == 0 {
		text := "Мастер пока не добавил услуги.\nНапишите ему напрямую 💬"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_menu"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID, text, keyboard)
		return
	}

	// Check if we need categories (9+ services)
	cats, _ := h.repos.Service.GetCategories(ctx, h.inst.Master.ID)
	if len(cats) > 0 {
		h.showCategories(ctx, chatID, cats)
		return
	}

	h.showServiceList(ctx, chatID, services)
}

func (h *Handler) showCategories(ctx context.Context, chatID int64, cats []*models.ServiceCategory) {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, cat := range cats {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(cat.Name, fmt.Sprintf("cat_%d", cat.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_menu"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.inst.SendWithInlineKeyboard(chatID, "Выберите категорию 👇", keyboard)
}

func (h *Handler) showServiceList(ctx context.Context, chatID int64, services []*models.Service) {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, svc := range services {
		priceStr := fmt.Sprintf("%d ₸", svc.Price)
		if svc.PriceFrom {
			priceStr = fmt.Sprintf("от %d ₸", svc.Price)
		}
		dur := formatDuration(svc.DurationMin)
		label := fmt.Sprintf("%s — %s — %s", svc.Name, priceStr, dur)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("svc_%d", svc.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_menu"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.inst.SendWithInlineKeyboard(chatID, "Выберите услугу 👇", keyboard)
}

func (h *Handler) handleCategorySelected(ctx context.Context, chatID int64, userID int64, data string) {
	catID, _ := strconv.Atoi(strings.TrimPrefix(data, "cat_"))
	services, err := h.repos.Service.GetActive(ctx, h.inst.Master.ID)
	if err != nil {
		return
	}
	var filtered []*models.Service
	for _, s := range services {
		if s.CategoryID != nil && *s.CategoryID == catID {
			filtered = append(filtered, s)
		}
	}
	h.showServiceList(ctx, chatID, filtered)
}

// ── Step 2: Service selected → show calendar ──────────────────────────────

func (h *Handler) handleServiceSelected(ctx context.Context, chatID int64, userID int64, data string) {
	svcID, _ := strconv.Atoi(strings.TrimPrefix(data, "svc_"))
	session := h.inst.GetSession(userID)
	session.Step = models.StepSelectDate
	session.ServiceID = svcID
	h.inst.SetSession(userID, session)

	h.showCalendar(ctx, chatID, userID)
}

func (h *Handler) showCalendar(ctx context.Context, chatID int64, userID int64) {
	master := h.inst.Master
	now := time.Now()
	minBooking := now.Add(time.Duration(master.MinHoursBeforeBooking) * time.Hour)

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for i := 0; i < 14; i++ {
		date := now.AddDate(0, 0, i)
		if date.Before(minBooking) && i == 0 {
			// Check if today still has slots after minBooking
		}

		// Check if day is working
		if !h.isDayWorking(master, date.Weekday()) {
			continue
		}

		// Check if day has any free slots
		hasFree, _ := h.dayHasFreeSlots(ctx, master.ID, date, userID)
		if !hasFree {
			continue
		}

		label := formatDate(date)
		cbData := fmt.Sprintf("date_%s", date.Format("2006-01-02"))
		btn := tgbotapi.NewInlineKeyboardButtonData(label, cbData)
		row = append(row, btn)

		if len(row) == 2 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}

	if len(rows) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Написать мастеру", "back_to_menu"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID,
			"К сожалению, свободных окон нет 😔\nНапишите мастеру напрямую.", keyboard)
		return
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_services"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.inst.SendWithInlineKeyboard(chatID, "Выберите день 📅", keyboard)
}

// ── Step 3: Date selected → show time slots ───────────────────────────────

func (h *Handler) handleDateSelected(ctx context.Context, chatID int64, userID int64, data string) {
	dateStr := strings.TrimPrefix(data, "date_")
	session := h.inst.GetSession(userID)
	session.Step = models.StepSelectTime
	session.Date = dateStr
	h.inst.SetSession(userID, session)

	h.showTimeSlots(ctx, chatID, userID, dateStr)
}

func (h *Handler) showTimeSlots(ctx context.Context, chatID int64, userID int64, dateStr string) {
	master := h.inst.Master
	session := h.inst.GetSession(userID)

	svc, err := h.repos.Service.GetByID(ctx, session.ServiceID)
	if err != nil {
		return
	}

	date, _ := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	slots := h.getAvailableSlots(ctx, master, date, svc.DurationMin)

	if len(slots) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("← Выбрать другой день", "back_to_services"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID,
			"На этот день нет свободных слотов 😔\nВыберите другой день.", keyboard)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, slot := range slots {
		label := slot.Format("15:04")
		cbData := fmt.Sprintf("time_%s", slot.Format("15:04"))
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
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_services"),
	))

	parsedDate, _ := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	h.inst.SendWithInlineKeyboard(chatID,
		fmt.Sprintf("<b>%s</b> — выберите время ⏰", formatDateFull(parsedDate)),
		tgbotapi.NewInlineKeyboardMarkup(rows...))
}

// ── Step 4: Time selected → collect name/phone if missing ────────────────

func (h *Handler) handleTimeSelected(ctx context.Context, chatID int64, userID int64, data string) {
	timeStr := strings.TrimPrefix(data, "time_")
	session := h.inst.GetSession(userID)

	// Store time in date field temporarily as "date time"
	session.Date = session.Date + " " + timeStr
	h.inst.SetSession(userID, session)

	client, _ := h.repos.Client.GetByTelegramID(ctx, h.inst.Master.ID, userID)

	if client.Name == "" {
		session.Step = models.StepAwaitName
		h.inst.SetSession(userID, session)
		h.inst.SendMessage(chatID, "Как вас зовут? Введите ваше имя:")
		return
	}

	if client.Phone == "" {
		session.Step = models.StepAwaitPhone
		h.inst.SetSession(userID, session)
		h.sendPhoneRequest(chatID)
		return
	}

	h.showBookingConfirmation(ctx, chatID, userID, client)
}

func (h *Handler) handleNameInput(ctx context.Context, msg *tgbotapi.Message, client *models.Client, session *models.SessionState) {
	name := strings.TrimSpace(msg.Text)
	if name == "" {
		h.inst.SendMessage(msg.Chat.ID, "Пожалуйста, введите ваше имя:")
		return
	}

	h.repos.Client.UpdateNamePhone(ctx, client.ID, name, client.Phone)
	client.Name = name

	if client.Phone == "" {
		session.Step = models.StepAwaitPhone
		h.inst.SetSession(msg.From.ID, session)
		h.sendPhoneRequest(msg.Chat.ID)
		return
	}

	h.showBookingConfirmation(ctx, msg.Chat.ID, msg.From.ID, client)
}

func (h *Handler) sendPhoneRequest(chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 Отправить мой номер"),
		),
	)
	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = true
	h.inst.SendWithReplyKeyboard(chatID,
		"Ваш номер телефона?\n\nНажмите кнопку или введите вручную:", keyboard)
}

func (h *Handler) handlePhoneInput(ctx context.Context, msg *tgbotapi.Message, client *models.Client, session *models.SessionState) {
	var phone string
	if msg.Contact != nil {
		phone = msg.Contact.PhoneNumber
	} else {
		phone = strings.TrimSpace(msg.Text)
	}

	if phone == "" {
		h.inst.SendMessage(msg.Chat.ID, "Пожалуйста, введите номер телефона:")
		return
	}

	h.repos.Client.UpdateNamePhone(ctx, client.ID, client.Name, phone)
	client.Phone = phone

	session.Step = models.StepConfirmBooking
	h.inst.SetSession(msg.From.ID, session)

	h.showBookingConfirmation(ctx, msg.Chat.ID, msg.From.ID, client)
}

// ── Step 5: Show summary and confirm ─────────────────────────────────────

func (h *Handler) showBookingConfirmation(ctx context.Context, chatID int64, userID int64, client *models.Client) {
	session := h.inst.GetSession(userID)

	svc, err := h.repos.Service.GetByID(ctx, session.ServiceID)
	if err != nil {
		return
	}

	parts := strings.SplitN(session.Date, " ", 2)
	if len(parts) != 2 {
		return
	}
	dateStr, timeStr := parts[0], parts[1]

	startsAt, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, time.Local)
	if err != nil {
		return
	}

	priceStr := fmt.Sprintf("%d ₸", svc.Price)
	if svc.PriceFrom {
		priceStr = "от " + priceStr
	}

	text := fmt.Sprintf(
		"Проверьте вашу запись ✅\n\n"+
			"👤 <b>%s</b>\n"+
			"📱 %s\n"+
			"💅 %s\n"+
			"📅 %s\n"+
			"⏰ %s (%s)\n"+
			"💰 %s",
		client.Name, client.Phone,
		svc.Name,
		formatDateFull(startsAt),
		startsAt.Format("15:04"), formatDuration(svc.DurationMin),
		priceStr,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", "confirm_booking"),
			tgbotapi.NewInlineKeyboardButtonData("← Изменить", "back_to_services"),
		),
	)

	// Remove reply keyboard
	removeKb := tgbotapi.NewRemoveKeyboard(true)
	rm := tgbotapi.NewMessage(chatID, "⏳")
	rm.ReplyMarkup = removeKb
	h.inst.API.Send(rm)

	session.Step = models.StepConfirmBooking
	h.inst.SetSession(userID, session)

	h.inst.SendWithInlineKeyboard(chatID, text, keyboard)
}

// ── Step 6: Create booking ─────────────────────────────────────────────────

func (h *Handler) handleConfirmBooking(ctx context.Context, chatID int64, userID int64, client *models.Client) {
	session := h.inst.GetSession(userID)

	svc, err := h.repos.Service.GetByID(ctx, session.ServiceID)
	if err != nil {
		h.inst.SendMessage(chatID, "Что-то пошло не так. Попробуйте снова.")
		return
	}

	parts := strings.SplitN(session.Date, " ", 2)
	dateStr, timeStr := parts[0], parts[1]
	startsAt, _ := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, time.Local)
	endsAt := startsAt.Add(time.Duration(svc.DurationMin) * time.Minute)

	// Check slot still free
	taken, err := h.repos.Booking.IsSlotTaken(ctx, h.inst.Master.ID, startsAt, endsAt)
	if err != nil || taken {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 Выбрать другое время", "back_to_services"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID,
			"Это время только что заняли 😔\nВыберите другое время.", keyboard)
		return
	}

	bookingID, err := h.repos.Booking.Create(ctx, &models.Booking{
		MasterID:  h.inst.Master.ID,
		ClientID:  client.ID,
		ServiceID: svc.ID,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
	})
	if err != nil {
		log.Printf("Create booking error: %v", err)
		h.inst.SendMessage(chatID, "Ошибка при создании записи. Попробуйте позже.")
		return
	}

	h.inst.ClearSession(userID)

	// Notify client
	h.inst.SendMessage(chatID,
		"Отлично! Заявка отправлена 🎉\nОжидайте подтверждения от мастера.\nОбычно это занимает несколько минут.")
	h.sendMainMenu(ctx, chatID, "")

	// Notify master via admin bot
	h.notifyMasterNewBooking(ctx, bookingID, svc, client, startsAt)
}

func (h *Handler) notifyMasterNewBooking(ctx context.Context, bookingID int, svc *models.Service, client *models.Client, startsAt time.Time) {
	master := h.inst.Master

	booking := &models.Booking{
		ID:               bookingID,
		MasterID:         master.ID,
		ClientName:       client.Name,
		ClientPhone:      client.Phone,
		ClientTelegramID: client.TelegramID,
		ServiceName:      svc.Name,
		ServicePrice:     svc.Price,
		StartsAt:         startsAt,
	}

	// Send via Notifier interface → Manager → AdminBot
	h.inst.Notifier.NotifyMasterNewBooking(master.ID, master.MasterTelegramID, booking)
}

// ── Cancellation ──────────────────────────────────────────────────────────

func (h *Handler) handleMyBookings(ctx context.Context, msg *tgbotapi.Message, client *models.Client) {
	bookings, err := h.repos.Booking.GetUpcomingForClient(ctx, h.inst.Master.ID, client.ID)
	if err != nil || len(bookings) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 Записаться", "booking_start"),
			),
		)
		h.inst.SendWithInlineKeyboard(msg.Chat.ID,
			"У вас нет предстоящих записей.", keyboard)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, b := range bookings {
		label := fmt.Sprintf("%s — %s", formatDateFull(b.StartsAt), b.ServiceName)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("noop_%d", b.ID)),
		})
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", fmt.Sprintf("cancel_booking_%d", b.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Перенести", fmt.Sprintf("reschedule_%d", b.ID)),
		})
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.inst.SendWithInlineKeyboard(msg.Chat.ID, "Ваши записи:", keyboard)
}

func (h *Handler) handleCancelBookingPrompt(ctx context.Context, chatID int64, userID int64, data string) {
	bookingIDStr := strings.TrimPrefix(data, "cancel_booking_")
	bookingID, _ := strconv.Atoi(bookingIDStr)

	booking, err := h.repos.Booking.GetByID(ctx, bookingID)
	if err != nil {
		return
	}

	master := h.inst.Master
	limitTime := booking.StartsAt.Add(-time.Duration(master.CancelLimitHours) * time.Hour)
	if time.Now().After(limitTime) {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Написать мастеру", "back_to_menu"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID,
			fmt.Sprintf("К сожалению, отменить запись можно не позже чем за %d часов 😔\n\nЕсли возникла срочная ситуация — напишите мастеру напрямую.", master.CancelLimitHours),
			keyboard)
		return
	}

	session := h.inst.GetSession(userID)
	session.BookingID = bookingID
	h.inst.SetSession(userID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("😔 Не смогу прийти", fmt.Sprintf("cancel_reason_%d_not_coming", bookingID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Хочу другое время", fmt.Sprintf("cancel_reason_%d_other_time", bookingID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Финансовые причины", fmt.Sprintf("cancel_reason_%d_financial", bookingID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_menu"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID,
		fmt.Sprintf("Укажите причину отмены:\n\n💅 %s\n📅 %s — %s",
			booking.ServiceName, formatDateFull(booking.StartsAt), booking.StartsAt.Format("15:04")),
		keyboard)
}

func (h *Handler) handleCancelReason(ctx context.Context, chatID int64, userID int64, data string) {
	// data: cancel_reason_{bookingID}_{reason}
	parts := strings.SplitN(strings.TrimPrefix(data, "cancel_reason_"), "_", 2)
	if len(parts) != 2 {
		return
	}
	bookingID, _ := strconv.Atoi(parts[0])
	reason := parts[1]

	reasonText := map[string]string{
		"not_coming": "Не смогу прийти",
		"other_time": "Хочу другое время",
		"financial":  "Финансовые причины",
	}[reason]
	booking, err := h.repos.Booking.GetByID(ctx, bookingID)
	if err != nil {
		return
	}
	h.repos.Booking.Cancel(ctx, bookingID, models.StatusCancelledByClient, reasonText)
	h.inst.Notifier.NotifyMasterClientCancelled(h.inst.Master.ID, h.inst.Master.MasterTelegramID, booking, reasonText)
	h.inst.ClearSession(userID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Записаться снова", "booking_start"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID,
		"Запись отменена 😔\n\nБудем рады видеть вас снова!", keyboard)
}

func (h *Handler) handleReschedule(ctx context.Context, chatID int64, userID int64, data string, client *models.Client) {
	bookingIDStr := strings.TrimPrefix(data, "reschedule_")
	bookingID, _ := strconv.Atoi(bookingIDStr)

	booking, _ := h.repos.Booking.GetByID(ctx, bookingID)

	session := h.inst.GetSession(userID)
	session.ServiceID = booking.ServiceID
	session.BookingID = bookingID
	session.Step = models.StepSelectDate
	h.inst.SetSession(userID, session)

	// Cancel old booking
	h.repos.Booking.Cancel(ctx, bookingID, models.StatusCancelledByClient, "Перенос")

	h.showCalendar(ctx, chatID, userID)
}

// ── Reviews ───────────────────────────────────────────────────────────────

func (h *Handler) handleLeaveReview(ctx context.Context, msg *tgbotapi.Message, client *models.Client) {
	h.handleLeaveReviewCallback(ctx, msg.Chat.ID, msg.From.ID, client)
}

func (h *Handler) handleLeaveReviewCallback(ctx context.Context, chatID int64, userID int64, client *models.Client) {
	session := h.inst.GetSession(userID)
	session.Step = models.StepAwaitReview
	h.inst.SetSession(userID, session)
	h.inst.SendMessage(chatID, "Напишите ваш отзыв ✏️\n\nМастер обязательно прочитает его 🤍")
}

func (h *Handler) handleReviewInput(ctx context.Context, msg *tgbotapi.Message, client *models.Client, session *models.SessionState) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	bookingID := session.BookingID
	if bookingID == 0 {
		h.inst.SendMessage(msg.Chat.ID, "Ошибка: запись для отзыва не найдена")
		return
	}

	err := h.repos.Review.Create(ctx, h.inst.Master.ID, client.ID, &bookingID, text)
	if err != nil {
		log.Printf("Failed to save review: %v", err)
		h.inst.SendMessage(msg.Chat.ID, "Произошла ошибка при сохранении отзыва. Попробуйте снова.")
		return
	}

	booking, err := h.repos.Booking.GetByID(ctx, bookingID)
	if err != nil {
		log.Printf("GetByID booking error: %v", err)
		return
	}

	service, err := h.repos.Service.GetByID(ctx, booking.ServiceID)
	if err != nil {
		return
	}

	h.inst.Notifier.NotifyMasterNewReview(h.inst.Master.ID, h.inst.Master.MasterTelegramID, client.Name, service.Name, text)

	h.inst.ClearSession(msg.From.ID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_to_menu"),
		),
	)
	h.inst.SendWithInlineKeyboard(msg.Chat.ID,
		"Спасибо за отзыв! 🙏", keyboard)
}

// ── Consent ───────────────────────────────────────────────────────────────

func (h *Handler) handleConsentAccept(ctx context.Context, chatID int64, userID int64, client *models.Client) {
	h.repos.Client.SaveConsent(ctx, client.ID)
	h.sendMainMenu(ctx, chatID, "Отлично! Выберите что вас интересует 👇")
}

// ── Helpers ───────────────────────────────────────────────────────────────

func (h *Handler) getAvailableSlots(ctx context.Context, master *models.Master, date time.Time, durationMin int) []time.Time {
	daySchedule := h.getDaySchedule(master, date.Weekday())
	if daySchedule.Start == nil || daySchedule.End == nil {
		return nil
	}

	startTime := *daySchedule.Start
	endTime := *daySchedule.End

	workStart := time.Date(date.Year(), date.Month(), date.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
	workEnd := time.Date(date.Year(), date.Month(), date.Day(),
		endTime.Hour(), endTime.Minute(), 0, 0, time.Local)

	minBookingTime := time.Now().Add(time.Duration(master.MinHoursBeforeBooking) * time.Hour)
	interval := time.Duration(master.SlotIntervalMin) * time.Minute
	svcDuration := time.Duration(durationMin) * time.Minute

	var slots []time.Time
	for t := workStart; t.Add(svcDuration).Before(workEnd) || t.Add(svcDuration).Equal(workEnd); t = t.Add(interval) {
		if t.Before(minBookingTime) {
			continue
		}
		taken, _ := h.repos.Booking.IsSlotTaken(ctx, master.ID, t, t.Add(svcDuration))
		if !taken {
			slots = append(slots, t)
		}
	}
	return slots
}

func (h *Handler) isDayWorking(master *models.Master, weekday time.Weekday) bool {
	sched := h.getDaySchedule(master, weekday)
	return sched.Start != nil && sched.End != nil
}

func (h *Handler) getDaySchedule(master *models.Master, weekday time.Weekday) models.DaySchedule {
	switch weekday {
	case time.Monday:
		return master.Schedule.Mon
	case time.Tuesday:
		return master.Schedule.Tue
	case time.Wednesday:
		return master.Schedule.Wed
	case time.Thursday:
		return master.Schedule.Thu
	case time.Friday:
		return master.Schedule.Fri
	case time.Saturday:
		return master.Schedule.Sat
	case time.Sunday:
		return master.Schedule.Sun
	}
	return models.DaySchedule{}
}

func (h *Handler) dayHasFreeSlots(ctx context.Context, masterID int, date time.Time, userID int64) (bool, error) {
	master := h.inst.Master
	session := h.inst.GetSession(userID)
	svcDuration := 60 // default
	if session.ServiceID > 0 {
		svc, err := h.repos.Service.GetByID(ctx, session.ServiceID)
		if err == nil {
			svcDuration = svc.DurationMin
		}
	}
	slots := h.getAvailableSlots(ctx, master, date, svcDuration)
	return len(slots) > 0, nil
}

func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d мин", minutes)
	}
	h := minutes / 60
	m := minutes % 60
	if m == 0 {
		return fmt.Sprintf("%d ч", h)
	}
	return fmt.Sprintf("%d ч %d мин", h, m)
}

func formatDate(t time.Time) string {
	days := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	months := []string{"", "янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек"}
	return fmt.Sprintf("%s %d %s", days[t.Weekday()], t.Day(), months[t.Month()])
}

func formatDateFull(t time.Time) string {
	days := []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%s, %d %s", strings.Title(days[t.Weekday()]), t.Day(), months[t.Month()])
}
