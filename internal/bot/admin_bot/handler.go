package admin_bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"beauty-bot/internal/types"
	"beauty-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	inst  *types.BotInstance
	repos *types.Repos
}

func NewHandler(inst *types.BotInstance, repos *types.Repos) *Handler {
	return &Handler{inst: inst, repos: repos}
}

func (h *Handler) Handle(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}
	if update.Message == nil {
		return
	}

	msg := update.Message
	masterTgID := h.inst.Master.ID // This is DB id — we need to verify by TG ID

	// Only master can use admin bot
	// We store master's telegram ID separately — for now trust anyone who has the token
	_ = masterTgID

	session := h.inst.GetSession(msg.From.ID)

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			h.sendMainMenu(ctx, msg.Chat.ID)
		}
		return
	}

	// Handle session steps
	switch session.Step {
	case models.StepAwaitBroadcastMsg:
		h.handleBroadcastMessage(ctx, msg, session)
	default:
		switch msg.Text {
		case "📅 Расписание":
			h.handleScheduleToday(ctx, msg.Chat.ID)
		case "💅 Услуги":
			h.handleServices(ctx, msg.Chat.ID)
		case "👥 Клиенты":
			h.handleClients(ctx, msg.Chat.ID)
		case "⭐ Отзывы":
			h.handleReviews(ctx, msg.Chat.ID)
		case "🕐 График работы":
			h.handleWorkSchedule(ctx, msg.Chat.ID)
		case "📢 Рассылка":
			h.handleBroadcast(ctx, msg.Chat.ID)
		case "📊 Статистика":
			h.handleStats(ctx, msg.Chat.ID)
		default:
			h.sendMainMenu(ctx, msg.Chat.ID)
		}
	}
}

func (h *Handler) sendMainMenu(ctx context.Context, chatID int64) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 Расписание"),
			tgbotapi.NewKeyboardButton("💅 Услуги"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👥 Клиенты"),
			tgbotapi.NewKeyboardButton("⭐ Отзывы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🕐 График работы"),
			tgbotapi.NewKeyboardButton("📢 Рассылка"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
	)
	keyboard.ResizeKeyboard = true
	h.inst.SendWithReplyKeyboard(chatID, "Панель управления 🎛", keyboard)
}

// ── Schedule ──────────────────────────────────────────────────────────────

func (h *Handler) handleScheduleToday(ctx context.Context, chatID int64) {
	bookings, err := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, time.Now())
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка загрузки расписания.")
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "sched_today"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра", "sched_tomorrow"),
			tgbotapi.NewInlineKeyboardButtonData("Выбрать день", "sched_select_month"),
		),
	)
	if len(bookings) == 0 {
		h.inst.SendWithInlineKeyboard(chatID, "На сегодня записей нет 🌿", keyboard)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📅 %s</b>\n\n", formatDateFull(time.Now())))
	total := 0
	for _, b := range bookings {
		sb.WriteString(fmt.Sprintf(
			"⏰ <b>%s</b> — %s\n   💅 %s\n   📱 %s\n\n",
			b.StartsAt.Format("15:04"),
			b.ClientName, b.ServiceName, b.ClientPhone,
		))
		total += b.ServicePrice
	}
	sb.WriteString(fmt.Sprintf("Итого: <b>%d записей</b> / ~%d ₸", len(bookings), total))

	h.inst.SendWithInlineKeyboard(chatID, sb.String(), keyboard)
}

// ── Services ──────────────────────────────────────────────────────────────

func (h *Handler) handleServices(ctx context.Context, chatID int64) {
	services, _ := h.repos.Service.GetAll(ctx, h.inst.Master.ID)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range services {
		status := "✅"
		if !s.IsActive {
			status = "🚫"
		}
		label := fmt.Sprintf("%s %s — %d ₸", status, s.Name, s.Price)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("admin_svc_%d", s.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить услугу", "admin_svc_add"),
	))

	if len(services) == 0 {
		h.inst.SendWithInlineKeyboard(chatID, "Услуги не добавлены.\nНажмите ➕ чтобы добавить.",
			tgbotapi.NewInlineKeyboardMarkup(rows...))
		return
	}

	h.inst.SendWithInlineKeyboard(chatID, "Ваши услуги 💅",
		tgbotapi.NewInlineKeyboardMarkup(rows...))
}

// ── Clients ───────────────────────────────────────────────────────────────

func (h *Handler) handleClients(ctx context.Context, chatID int64) {
	clients, _ := h.repos.Client.GetAllForMaster(ctx, h.inst.Master.ID)

	text := fmt.Sprintf("👥 <b>База клиентов</b> — %d человек\n\n", len(clients))
	if len(clients) == 0 {
		text += "Клиентов пока нет.\nПоделитесь ботом с клиентами! 🤍"
	} else {
		for i, c := range clients {
			if i >= 10 {
				text += fmt.Sprintf("\n...и ещё %d клиентов", len(clients)-10)
				break
			}
			text += fmt.Sprintf("👤 <b>%s</b> — %s\n", c.Name, c.Phone)
		}
	}
	h.inst.SendMessage(chatID, text)
}

// ── Reviews ───────────────────────────────────────────────────────────────

func (h *Handler) handleReviews(ctx context.Context, chatID int64) {
	reviews, err := h.repos.Review.GetAllForMaster(ctx, h.inst.Master.ID)
	if err != nil {
		log.Printf("Error fetching reviews: %v", err)
		h.inst.SendMessage(chatID, "Произошла ошибка при получении отзывов. Попробуйте снова.")
		return
	}

	if len(reviews) == 0 {
		h.inst.SendMessage(chatID, "Отзывов пока нет ⭐")
		return
	}

	// Показываем первую страницу
	h.sendReviewsPage(ctx, chatID, reviews, 0)
}
func (h *Handler) sendReviewsPage(ctx context.Context, chatID int64, reviews []*models.Review, page int) {
	const pageSize = 5

	start := page * pageSize
	end := start + pageSize
	if end > len(reviews) {
		end = len(reviews)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⭐ <b>Отзывы (%d)</b>\n\n", len(reviews)))

	months := map[time.Month]string{
		time.January: "янв", time.February: "фев", time.March: "мар",
		time.April: "апр", time.May: "май", time.June: "июн",
		time.July: "июл", time.August: "авг", time.September: "сен",
		time.October: "окт", time.November: "ноя", time.December: "дек",
	}

	for _, r := range reviews[start:end] {
		date := fmt.Sprintf("%02d %s %02d:%02d",
			r.CreatedAt.Day(),
			months[r.CreatedAt.Month()],
			r.CreatedAt.Hour(),
			r.CreatedAt.Minute(),
		)
		sb.WriteString(fmt.Sprintf(
			"👤 <b>%s</b>\n🕒 %s\n💬 %s\n──────────────\n",
			r.ClientName, date, r.Text,
		))
	}

	// Кнопки для пагинации
	var buttons [][]tgbotapi.InlineKeyboardButton
	if start > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("reviews_page_%d", page-1)),
		))
	}
	if end < len(reviews) {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("reviews_page_%d", page+1)),
		))
	}

	h.inst.SendWithInlineKeyboard(chatID, sb.String(), tgbotapi.NewInlineKeyboardMarkup(buttons...))
}

// ── Work schedule ─────────────────────────────────────────────────────────

func (h *Handler) handleWorkSchedule(ctx context.Context, chatID int64) {
	master := h.inst.Master
	s := master.Schedule

	dayName := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	schedDays := []models.DaySchedule{s.Sun, s.Mon, s.Tue, s.Wed, s.Thu, s.Fri, s.Sat}

	var sb strings.Builder
	sb.WriteString("🕐 <b>Ваш рабочий график:</b>\n\n")
	for i, day := range schedDays {
		if day.Start != nil && day.End != nil {
			sb.WriteString(fmt.Sprintf("%s: %s — %s ✅\n", dayName[i], *day.Start, *day.End))
		} else {
			sb.WriteString(fmt.Sprintf("%s: выходной ❌\n", dayName[i]))
		}
	}
	sb.WriteString(fmt.Sprintf(
		"\nИнтервал: <b>%d мин</b>\nМин. до записи: <b>%d ч</b>\nОтмена не позднее: <b>%d ч</b>",
		master.SlotIntervalMin, master.MinHoursBeforeBooking, master.CancelLimitHours,
	))

	h.inst.SendMessage(chatID, sb.String())
}

// ── Broadcast ─────────────────────────────────────────────────────────────

func (h *Handler) handleBroadcast(ctx context.Context, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Не были 1 месяц", "broadcast_1m"),
			tgbotapi.NewInlineKeyboardButtonData("Не были 2 месяца", "broadcast_2m"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Не были 3 месяца", "broadcast_3m"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, "📢 Выберите сегмент для рассылки:", keyboard)
}

func (h *Handler) handleBroadcastSegment(ctx context.Context, chatID int64, userID int64, months int) {
	inactiveSince := time.Now().AddDate(0, -months, 0)
	clients, _ := h.repos.Client.GetAllForBroadcast(ctx, h.inst.Master.ID, inactiveSince)

	if len(clients) == 0 {
		h.inst.SendMessage(chatID, "Клиентов в этой категории нет.")
		return
	}

	session := h.inst.GetSession(userID)
	session.Step = models.StepAwaitBroadcastMsg
	h.inst.SetSession(userID, session)

	// Show templates
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💌 Напоминание о себе", "tmpl_reminder"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Акция", "tmpl_promo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✨ Новая услуга", "tmpl_new_service"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID,
		fmt.Sprintf("Клиентов в сегменте: <b>%d</b>\n\nВыберите шаблон или напишите сообщение:", len(clients)),
		keyboard)
}

func (h *Handler) handleBroadcastMessage(ctx context.Context, msg *tgbotapi.Message, session *models.SessionState) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"✅ Отправить", fmt.Sprintf("broadcast_send_%s", text[:min(len(text), 50)])),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "broadcast_edit"),
		),
	)

	preview := fmt.Sprintf("Предпросмотр:\n\n<i>%s</i>", text)
	h.inst.SendWithInlineKeyboard(msg.Chat.ID, preview, keyboard)
}

// ── Stats ─────────────────────────────────────────────────────────────────

func (h *Handler) handleStats(ctx context.Context, chatID int64) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)

	total, completed, cancelled, revenue, err := h.repos.Booking.GetStatsForMaster(
		ctx, h.inst.Master.ID, monthStart, monthEnd)
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка загрузки статистики.")
		return
	}

	clients, _ := h.repos.Client.GetAllForMaster(ctx, h.inst.Master.ID)
	reviews, _ := h.repos.Review.GetAllForMaster(ctx, h.inst.Master.ID)

	months := []string{"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
		"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}

	text := fmt.Sprintf(
		"📊 <b>Статистика за %s</b>\n\n"+
			"📅 Записей всего: <b>%d</b>\n"+
			"✅ Состоялось: <b>%d</b>\n"+
			"❌ Отменено: <b>%d</b>\n\n"+
			"💰 Выручка: <b>~%d ₸</b>\n\n"+
			"👥 Клиентов всего: <b>%d</b>\n"+
			"⭐ Отзывов: <b>%d</b>",
		months[now.Month()],
		total, completed, cancelled,
		revenue,
		len(clients),
		len(reviews),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Эта неделя", "stats_week"),
			tgbotapi.NewInlineKeyboardButtonData("Этот месяц", "stats_month"),
			tgbotapi.NewInlineKeyboardButtonData("Всего", "stats_all"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, text, keyboard)
}

// ── Callbacks ─────────────────────────────────────────────────────────────

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.inst.API.Send(tgbotapi.NewCallback(cb.ID, ""))

	data := cb.Data
	chatID := cb.Message.Chat.ID
	userID := cb.From.ID

	switch {
	case data == "sched_today":
		h.handleScheduleToday(ctx, chatID)
	case data == "sched_tomorrow":
		bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, time.Now().AddDate(0, 0, 1))
		h.showDaySchedule(ctx, chatID, time.Now().AddDate(0, 0, 1), bookings)
	case data == "sched_select_month":
	now := time.Now()
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 4; i++ {
		m := now.AddDate(0, i, 0)
		monthName := fmt.Sprintf("%s %d", m.Month().String(), m.Year()) // для пользователя
		callbackData := fmt.Sprintf("sched_select_day_%d-%02d", m.Year(), m.Month()) // для обработки
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(monthName, callbackData),
		))
	}
	h.inst.SendWithInlineKeyboard(chatID, "Выберите месяц:", tgbotapi.NewInlineKeyboardMarkup(rows...))
	case strings.HasPrefix(data, "sched_select_day_"):
	parts := strings.Split(strings.TrimPrefix(data, "sched_select_day_"), "-")
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])

	// Генерируем дни выбранного месяца
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
	var rows [][]tgbotapi.InlineKeyboardButton
	for day := 1; day <= daysInMonth; day++ {
		callbackData := fmt.Sprintf("sched_day_%d-%02d-%02d", year, month, day)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%02d", day), callbackData),
		))
	}
	h.inst.SendWithInlineKeyboard(chatID, "Выберите день:", tgbotapi.NewInlineKeyboardMarkup(rows...))
	case strings.HasPrefix(data, "sched_day_"):
	parts := strings.Split(strings.TrimPrefix(data, "sched_day_"), "-")
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, date)
	h.showDaySchedule(ctx, chatID, date, bookings)
	case strings.HasPrefix(data, "sched_day_"):
    offset, _ := strconv.Atoi(strings.TrimPrefix(data, "sched_day_"))
    day := time.Now().AddDate(0, 0, offset)
    bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, day)
    h.showDaySchedule(ctx, chatID, day, bookings)
	case data == "broadcast_1m":
		h.handleBroadcastSegment(ctx, chatID, userID, 1)
	case data == "broadcast_2m":
		h.handleBroadcastSegment(ctx, chatID, userID, 2)
	case data == "broadcast_3m":
		h.handleBroadcastSegment(ctx, chatID, userID, 3)
	case strings.HasPrefix(data, "tmpl_"):
		h.handleTemplate(ctx, chatID, userID, data)
	case strings.HasPrefix(data, "admin_confirm_"):
		bookingID, _ := strconv.Atoi(strings.TrimPrefix(data, "admin_confirm_"))
		h.repos.Booking.Confirm(ctx, bookingID, "master")
		h.inst.SendMessage(chatID, "✅ Запись подтверждена!")
		h.notifyClientConfirmed(ctx, bookingID)
	case strings.HasPrefix(data, "admin_reject_"):
		bookingID, _ := strconv.Atoi(strings.TrimPrefix(data, "admin_reject_"))
		h.repos.Booking.Cancel(ctx, bookingID, models.StatusCancelledByMaster, "")
		h.inst.SendMessage(chatID, "❌ Запись отклонена.")
		h.notifyClientRejected(ctx, bookingID)
	case strings.HasPrefix(data, "admin_svc_"):
		svcIDStr := strings.TrimPrefix(data, "admin_svc_")
		if svcIDStr == "add" {
			h.inst.SendMessage(chatID, "Функция добавления услуги в разработке.\nОбратитесь к поддержке.")
			return
		}
		svcID, _ := strconv.Atoi(svcIDStr)
		h.showServiceActions(ctx, chatID, svcID)
		case strings.HasPrefix(data, "reviews_page_"):
	pageStr := strings.TrimPrefix(data, "reviews_page_")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return
	}
	reviews, err := h.repos.Review.GetAllForMaster(ctx, h.inst.Master.ID)
	if err != nil || len(reviews) == 0 {
		return
	}
	h.sendReviewsPage(ctx, chatID, reviews, page)
	case data == "stats_week", data == "stats_month", data == "stats_all":
		h.handleStats(ctx, chatID)
	}
}

func (h *Handler) showDaySchedule(ctx context.Context, chatID int64, date time.Time, bookings []*models.Booking) {
	if len(bookings) == 0 {
		h.inst.SendMessage(chatID, fmt.Sprintf("На %s записей нет 🌿", formatDateFull(date)))
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📅 %s</b>\n\n", formatDateFull(date)))
	total := 0
	for _, b := range bookings {
		sb.WriteString(fmt.Sprintf("⏰ <b>%s</b> — %s\n   💅 %s\n\n",
			b.StartsAt.Format("15:04"), b.ClientName, b.ServiceName))
		total += b.ServicePrice
	}
	sb.WriteString(fmt.Sprintf("Итого: <b>%d</b> / ~%d ₸", len(bookings), total))
	h.inst.SendMessage(chatID, sb.String())
}

func (h *Handler) showServiceActions(ctx context.Context, chatID int64, svcID int) {
	svc, err := h.repos.Service.GetByID(ctx, svcID)
	if err != nil {
		return
	}

	hasBookings, _ := h.repos.Service.HasActiveBookings(ctx, svcID)
	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("svc_edit_%d", svcID)),
	))
	if svc.IsActive {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Скрыть", fmt.Sprintf("svc_hide_%d", svcID)),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Показать", fmt.Sprintf("svc_show_%d", svcID)),
		))
	}
	if !hasBookings {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("svc_delete_%d", svcID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← Назад", "back_services"),
	))

	priceStr := fmt.Sprintf("%d ₸", svc.Price)
	if svc.PriceFrom {
		priceStr = "от " + priceStr
	}
	text := fmt.Sprintf("<b>%s</b>\n%s — %s", svc.Name, priceStr, formatDuration(svc.DurationMin))
	h.inst.SendWithInlineKeyboard(chatID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleTemplate(ctx context.Context, chatID int64, userID int64, data string) {
	templates := map[string]string{
		"tmpl_reminder":    "[Имя], давно вас не видела! 🤍\nБуду рада снова вас принять.\nЗапишитесь прямо здесь 👇",
		"tmpl_promo":       "[Имя], специально для вас — скидка 10% на этой неделе! 🎁\nУспейте записаться 👇",
		"tmpl_new_service": "[Имя], у меня появилась новая услуга!\nХотите попробовать? 🤍",
	}

	tmpl := templates[strings.TrimPrefix(data, "tmpl_")]
	if tmpl == "" {
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Отправить", fmt.Sprintf("broadcast_confirm_%d", userID)),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "broadcast_edit"),
		),
	)
	session := h.inst.GetSession(userID)
	session.Step = models.StepIdle
	h.inst.SetSession(userID, session)

	h.inst.SendWithInlineKeyboard(chatID,
		fmt.Sprintf("Предпросмотр:\n\n<i>%s</i>", tmpl), keyboard)
}

func (h *Handler) notifyClientConfirmed(ctx context.Context, bookingID int) {
	booking, err := h.repos.Booking.GetByID(ctx, bookingID)
	if err != nil {
		log.Printf("notifyClientConfirmed: GetByID error: %v", err)
		return
	}
	// Send via Notifier interface → Manager → ClientBot
	h.inst.Notifier.NotifyClientConfirmed(h.inst.Master.ID, booking)
}

func (h *Handler) notifyClientRejected(ctx context.Context, bookingID int) {
	booking, err := h.repos.Booking.GetByID(ctx, bookingID)
	if err != nil {
		log.Printf("notifyClientRejected: GetByID error: %v", err)
		return
	}
	// Send via Notifier interface → Manager → ClientBot
	h.inst.Notifier.NotifyClientRejected(h.inst.Master.ID, booking, "")
}

func formatDateFull(t time.Time) string {
	days := []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%s, %d %s", strings.Title(days[t.Weekday()]), t.Day(), months[t.Month()])
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
