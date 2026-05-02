package admin_bot

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"beauty-bot/internal/models"
	"beauty-bot/internal/types"

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
	case models.StepAwaitServiceName:
		h.handleNewServiceName(ctx, msg)
	case models.StepAwaitServicePrice:
		h.handleNewServicePrice(ctx, msg)
	case models.StepAwaitServiceDuration:
		h.handleNewServiceDuration(ctx, msg)
	case models.StepAwaitEditServiceName:
		h.handleEditServiceName(ctx, msg)
	case models.StepAwaitEditServicePrice:
		h.handleEditServicePrice(ctx, msg)
	case models.StepAwaitEditServiceDuration:
		h.handleEditServiceDuration(ctx, msg)
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
			h.handleStats(ctx, msg.Chat.ID, "month")
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
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📢 Рассылка"),
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
	h.showDaySchedule(ctx, chatID, time.Now(), bookings)
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

	total := len(clients)
	blocked := 0
	noBroadcast := 0
	for _, c := range clients {
		if c.IsBlocked {
			blocked++
		}
		if c.NoBroadcast {
			noBroadcast++
		}
	}

	text := fmt.Sprintf(
		"👥 <b>База клиентов</b>\n\n"+
			"Всего: <b>%d</b>\n"+
			"🚫 Заблокированных: <b>%d</b>\n"+
			"🔕 Отписанных от рассылки: <b>%d</b>",
		total, blocked, noBroadcast,
	)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Список клиентов", "clients_list:0"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔕 Отписанные", "clients_no_broadcast"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Заблокированные", "clients_blocked"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, text, kb)
}

func (h *Handler) handleClientsList(ctx context.Context, chatID int64, page int) {
	clients, _ := h.repos.Client.GetAllForMaster(ctx, h.inst.Master.ID)

	const pageSize = 5
	total := len(clients)
	start := page * pageSize
	if start >= total {
		start = 0
		page = 0
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	text := fmt.Sprintf("👥 <b>Клиенты</b> (стр. %d/%d)\n\n", page+1, (total+pageSize-1)/pageSize)
	for _, c := range clients[start:end] {
		lastVisit := "—"
		if c.LastVisitAt != nil {
			lastVisit = c.LastVisitAt.Format("02.01.2006")
		}
		flags := ""
		if c.IsBlocked {
			flags += " 🚫"
		}
		if c.NoBroadcast {
			flags += " 🔕"
		}

		text += fmt.Sprintf("👤 <b>%s</b>%s\n📞 %s\n✅ Визитов: %d | 📅 %s\n\n",
			c.Name, flags, c.Phone, c.VisitCount, lastVisit)
	}

	// Кнопки пагинации
	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("◀️", fmt.Sprintf("clients_list:%d", page-1)))
	}
	if end < total {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("▶️", fmt.Sprintf("clients_list:%d", page+1)))
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "clients_menu"),
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.inst.SendWithInlineKeyboard(chatID, text, kb)
}

// ── Reviews ───────────────────────────────────────────────────────────────

func (h *Handler) handleReviews(ctx context.Context, chatID int64) {
	reviews, err := h.repos.Review.GetAllForMaster(ctx, h.inst.Master.ID)

	if len(reviews) == 0 {
		h.inst.SendMessage(chatID, "Пока нет отзывов")
		return
	}
	if err != nil {
		log.Printf("Error fetching reviews: %v", err)
		h.inst.SendMessage(chatID, "Произошла ошибка при получении отзывов. Попробуйте снова.")
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
			sb.WriteString(fmt.Sprintf("%s: %s - %s ✅\n",
				dayName[i], day.Start.Format("15:04"), day.End.Format("15:04")))
		} else {
			sb.WriteString(fmt.Sprintf("%s: выходной ❌\n", dayName[i]))
		}
	}
	sb.WriteString(fmt.Sprintf(
		"\nИнтервал: <b>%d мин</b>\nМин. до записи: <b>%d ч</b>\nОтмена не позднее: <b>%d ч</b>",
		master.SlotIntervalMin, master.MinHoursBeforeBooking, master.CancelLimitHours,
	))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Заблокировать время", "block_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Активные блокировки", "block_list"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, sb.String(), kb)
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
	session.BroadcastMonths = months
	h.inst.SetSession(userID, session)

	h.inst.SendMessage(chatID,
		fmt.Sprintf("Клиентов в сегменте: <b>%d</b>\n\nНапишите текст рассылки:", len(clients)))
}

func (h *Handler) handleBroadcastMessage(ctx context.Context, msg *tgbotapi.Message, session *models.SessionState) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	session.BroadcastText = text
	h.inst.SetSession(msg.From.ID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Отправить", "broadcast_send"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "broadcast_edit"),
		),
	)
	h.inst.SendWithInlineKeyboard(msg.Chat.ID,
		fmt.Sprintf("Предпросмотр:\n\n<i>%s</i>", text), keyboard)
}

// ── Stats ─────────────────────────────────────────────────────────────────

func (h *Handler) handleStats(ctx context.Context, chatID int64, period string) {
	now := time.Now()
	var start, end time.Time
	var periodLabel string

	switch period {
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 0, 7)
		periodLabel = "эту неделю"
	case "all":
		start = time.Time{}
		end = now.AddDate(1, 0, 0)
		periodLabel = "всё время"
	default: // month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 1, 0)
		months := []string{"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
			"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}
		periodLabel = months[now.Month()]
	}

	total, completed, cancelled, revenue, err := h.repos.Booking.GetStatsForMaster(
		ctx, h.inst.Master.ID, start, end)
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка загрузки статистики.")
		return
	}

	clients, _ := h.repos.Client.GetAllForMaster(ctx, h.inst.Master.ID)
	reviews, _ := h.repos.Review.GetForPeriod(ctx, h.inst.Master.ID, start, end)
	avgCheck := 0
	if completed > 0 {
		avgCheck = int(revenue) / completed
	}

	text := fmt.Sprintf(
		"📊 <b>Статистика за %s</b>\n\n"+
			"📅 Записей всего: <b>%d</b>\n"+
			"✅ Состоялось: <b>%d</b>\n"+
			"❌ Отменено: <b>%d</b>\n\n"+
			"💰 Выручка: <b>%d ₸</b>\n\n"+
			"📈 Средний чек: <b>%d ₸</b>\n\n"+
			"👥 Клиентов всего: <b>%d</b>\n"+
			"⭐ Отзывов: <b>%d</b>",
		periodLabel,
		total, completed, cancelled,
		revenue, avgCheck,
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
		tomorrow := time.Now().AddDate(0, 0, 1)
		bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, tomorrow)
		h.showDaySchedule(ctx, chatID, tomorrow, bookings)
	case data == "sched_select_month":
		now := time.Now()
		months := []string{"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
			"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}
		var rows [][]tgbotapi.InlineKeyboardButton
		for i := 0; i < 4; i++ {
			m := now.AddDate(0, i, 0)
			monthName := fmt.Sprintf("%s %d", months[m.Month()], m.Year())
			callbackData := fmt.Sprintf("sched_select_day_%d-%02d", m.Year(), m.Month())
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(monthName, callbackData),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", "sched_today"),
		))
		h.inst.SendWithInlineKeyboard(chatID, "Выберите месяц:", tgbotapi.NewInlineKeyboardMarkup(rows...))
	case strings.HasPrefix(data, "sched_select_day_"):
		parts := strings.Split(strings.TrimPrefix(data, "sched_select_day_"), "-")
		year, _ := strconv.Atoi(parts[0])
		month, _ := strconv.Atoi(parts[1])

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
		dateStr := strings.TrimPrefix(data, "sched_day_")
		date, _ := time.Parse("2006-01-02", dateStr)
		bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, date)
		h.showDaySchedule(ctx, chatID, date, bookings)
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

		booking, err := h.repos.Booking.GetByID(ctx, bookingID)
		if err != nil {
			return
		}
		if booking.Status != models.StatusPending {
			h.inst.API.Send(tgbotapi.NewCallback(cb.ID, "Уже обработано"))
			return
		}

		h.repos.Booking.Confirm(ctx, bookingID, "master")
		h.notifyClientConfirmed(ctx, bookingID)

		newText := cb.Message.Text + "\n\n✅ Подтверждено"
		edit := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, newText)
		edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
		h.inst.API.Send(edit)

	case strings.HasPrefix(data, "admin_reject_"):
		bookingID, _ := strconv.Atoi(strings.TrimPrefix(data, "admin_reject_"))

		booking, err := h.repos.Booking.GetByID(ctx, bookingID)
		if err != nil {
			return
		}
		if booking.Status != models.StatusPending {
			h.inst.API.Send(tgbotapi.NewCallback(cb.ID, "Уже обработано"))
			return
		}

		h.repos.Booking.Cancel(ctx, bookingID, models.StatusCancelledByMaster, "")
		h.notifyClientRejected(ctx, bookingID)

		newText := cb.Message.Text + "\n\n❌ Отклонено"
		edit := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, newText)
		edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
		h.inst.API.Send(edit)
	case strings.HasPrefix(data, "svc_edit_name_"):
		svcID, _ := strconv.Atoi(strings.TrimPrefix(data, "svc_edit_name_"))
		session := h.inst.GetSession(userID)
		session.Step = models.StepAwaitEditServiceName
		session.ServiceID = svcID
		h.inst.SetSession(userID, session)
		h.inst.SendMessage(chatID, "Введите новое название:")

	case strings.HasPrefix(data, "svc_edit_price_"):
		svcID, _ := strconv.Atoi(strings.TrimPrefix(data, "svc_edit_price_"))
		session := h.inst.GetSession(userID)
		session.Step = models.StepAwaitEditServicePrice
		session.ServiceID = svcID
		h.inst.SetSession(userID, session)
		h.inst.SendMessage(chatID, "Введите новую цену в тенге (только число):")

	case strings.HasPrefix(data, "svc_edit_duration_"):
		svcID, _ := strconv.Atoi(strings.TrimPrefix(data, "svc_edit_duration_"))
		session := h.inst.GetSession(userID)
		session.Step = models.StepAwaitEditServiceDuration
		session.ServiceID = svcID
		h.inst.SetSession(userID, session)
		h.inst.SendMessage(chatID, "Введите продолжительность в минутах:")

	case strings.HasPrefix(data, "svc_edit_"):
		svcID, _ := strconv.Atoi(strings.TrimPrefix(data, "svc_edit_"))
		h.showServiceEditMenu(ctx, chatID, svcID)
	case strings.HasPrefix(data, "svc_delete_"):
		svcIDStr := strings.TrimPrefix(data, "svc_delete_")
		svcID, err := strconv.Atoi(svcIDStr)
		if err != nil {
			h.inst.SendMessage(chatID, "Ошибка при удалении.")
			return
		}
		err = h.repos.Service.Delete(ctx, svcID)
		if err != nil {
			h.inst.SendMessage(chatID, "Не удалось удалить услугу.")
			return
		}
		h.inst.SendMessage(chatID, "Услуга удалена ✅")
		h.handleServices(ctx, chatID)
	case data == "back_services":
		h.handleServices(ctx, chatID)
	case strings.HasPrefix(data, "admin_svc_"):
		svcIDStr := strings.TrimPrefix(data, "admin_svc_")
		if svcIDStr == "add" {
			session := h.inst.GetSession(userID)
			session.Step = models.StepAwaitServiceName
			h.inst.SetSession(userID, session)
			h.inst.SendMessage(chatID, "Введите название услуги:")
			return
		}
		svcID, _ := strconv.Atoi(svcIDStr)
		h.showServiceActions(ctx, chatID, svcID)
	case data == "reviews":
		h.handleReviews(ctx, chatID)
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
	case data == "stats_week":
		h.handleStats(ctx, chatID, "week")
	case data == "stats_month":
		h.handleStats(ctx, chatID, "month")
	case data == "stats_all":
		h.handleStats(ctx, chatID, "all")
	case strings.HasPrefix(data, "sched_pending_"):
		dateStr := strings.TrimPrefix(data, "sched_pending_")
		date, _ := time.Parse("2006-01-02", dateStr)
		bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, date)
		h.showPendingBookings(ctx, chatID, date, bookings)

	case strings.HasPrefix(data, "sched_cancelled_"):
		dateStr := strings.TrimPrefix(data, "sched_cancelled_")
		date, _ := time.Parse("2006-01-02", dateStr)
		bookings, _ := h.repos.Booking.GetForDay(ctx, h.inst.Master.ID, date)
		h.showCancelledBookings(ctx, chatID, date, bookings)
	case strings.HasPrefix(data, "master_complete_"):
		bookingID, _ := strconv.Atoi(strings.TrimPrefix(data, "master_complete_"))
		h.repos.Booking.MarkComplete(ctx, bookingID)

		booking, err := h.repos.Booking.GetByID(ctx, bookingID)
		if err != nil {
			return
		}

		if err := h.repos.Client.RecalcVisitStats(ctx, booking.ClientID); err != nil {
			log.Printf("Failed to recalc visit stats for client %d: %v", booking.ClientID, err)
		}

		newText := fmt.Sprintf(
			"⏰ <b>%s</b> — %s\n💅 %s\n📱 %s\n🏁 Завершена",
			booking.StartsAt.Format("15:04"),
			booking.ClientName, booking.ServiceName, booking.ClientPhone,
		)
		edit := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, newText)
		edit.ParseMode = "HTML"
		edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
		h.inst.API.Send(edit)
	case strings.HasPrefix(data, "master_cancel_"):
		bookingID, _ := strconv.Atoi(strings.TrimPrefix(data, "master_cancel_"))
		h.repos.Booking.Cancel(ctx, bookingID, models.StatusCancelledByMaster, "")

		booking, err := h.repos.Booking.GetByID(ctx, bookingID)
		if err != nil {
			return
		}

		newText := fmt.Sprintf(
			"⏰ <b>%s</b> — %s\n💅 %s\n📱 %s\n❌ Отменена мастером",
			booking.StartsAt.Format("15:04"),
			booking.ClientName, booking.ServiceName, booking.ClientPhone,
		)
		edit := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, newText)
		edit.ParseMode = "HTML"
		edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
		h.inst.API.Send(edit)
		h.notifyClientRejected(ctx, bookingID)

	case strings.HasPrefix(data, "clients_list:"):
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "clients_list:"))
		h.handleClientsList(ctx, chatID, page)

	case data == "clients_menu":
		h.handleClients(ctx, chatID)

	case data == "clients_blocked":
		h.handleClientsFiltered(ctx, chatID, "blocked")

	case data == "clients_no_broadcast":
		h.handleClientsFiltered(ctx, chatID, "no_broadcast")

	case data == "work_schedule":
		h.handleWorkSchedule(ctx, chatID)

	case data == "block_menu":
		h.handleBlockSlots(ctx, chatID)

	case data == "block_day":
		h.handleBlockDay(ctx, chatID, userID)

	case strings.HasPrefix(data, "block_day_confirm_"):
		dateStr := strings.TrimPrefix(data, "block_day_confirm_")
		h.handleBlockDayConfirm(ctx, chatID, dateStr)

	case data == "block_list":
		h.handleBlockList(ctx, chatID)

	case strings.HasPrefix(data, "block_delete_"):
		id, _ := strconv.Atoi(strings.TrimPrefix(data, "block_delete_"))
		h.handleBlockDelete(ctx, chatID, id)

	case data == "block_slot":
		h.handleBlockSlot(ctx, chatID)

	case strings.HasPrefix(data, "block_slot_day_"):
		dateStr := strings.TrimPrefix(data, "block_slot_day_")
		h.handleBlockSlotDay(ctx, chatID, dateStr)

	case strings.HasPrefix(data, "block_slot_confirm_"):
		parts := strings.SplitN(strings.TrimPrefix(data, "block_slot_confirm_"), "_", 2)
		if len(parts) == 2 {
			h.handleBlockSlotConfirm(ctx, chatID, parts[0], parts[1])
		}

	case data == "block_period":
		h.handleBlockPeriod(ctx, chatID)

	case strings.HasPrefix(data, "block_period_start_"):
		startStr := strings.TrimPrefix(data, "block_period_start_")
		h.handleBlockPeriodEnd(ctx, chatID, startStr)

	case strings.HasPrefix(data, "block_period_confirm_"):
		parts := strings.SplitN(strings.TrimPrefix(data, "block_period_confirm_"), "_", 2)
		if len(parts) == 2 {
			h.handleBlockPeriodConfirm(ctx, chatID, parts[0], parts[1])
		}
	case data == "broadcast_send":
		session := h.inst.GetSession(userID)
		if session.BroadcastText == "" {
			h.inst.SendMessage(chatID, "Текст рассылки не найден. Начните заново.")
			return
		}
		h.executeBroadcast(ctx, chatID, userID, session.BroadcastText, session.BroadcastMonths)
		session.BroadcastText = ""
		session.BroadcastMonths = 0
		h.inst.SetSession(userID, session)

	case data == "broadcast_edit":
		session := h.inst.GetSession(userID)
		session.Step = models.StepAwaitBroadcastMsg
		h.inst.SetSession(userID, session)
		h.inst.SendMessage(chatID, "Введите новый текст рассылки:")
	}
}

func (h *Handler) handleClientsFiltered(ctx context.Context, chatID int64, filter string) {
	clients, _ := h.repos.Client.GetAllForMaster(ctx, h.inst.Master.ID)

	var filtered []*models.Client
	var title string

	switch filter {
	case "blocked":
		title = "🚫 Заблокированные клиенты"
		for _, c := range clients {
			if c.IsBlocked {
				filtered = append(filtered, c)
			}
		}
	case "no_broadcast":
		title = "🔕 Отписанные от рассылки"
		for _, c := range clients {
			if c.NoBroadcast {
				filtered = append(filtered, c)
			}
		}
	}

	if len(filtered) == 0 {
		text := fmt.Sprintf("%s\n\nТаких клиентов нет.", title)
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "clients_menu"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID, text, kb)
		return
	}

	text := fmt.Sprintf("%s — %d\n\n", title, len(filtered))
	for _, c := range filtered {
		lastVisit := "—"
		if c.LastVisitAt != nil {
			lastVisit = c.LastVisitAt.Format("02.01.2006")
		}
		text += fmt.Sprintf("👤 <b>%s</b>\n📞 %s\n✅ Визитов: %d | 📅 %s\n\n",
			c.Name, c.Phone, c.VisitCount, lastVisit)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "clients_menu"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, text, kb)
}

func (h *Handler) showDaySchedule(ctx context.Context, chatID int64, date time.Time, bookings []*models.Booking) {
	// Сортируем по времени
	sort.Slice(bookings, func(i, j int) bool {
		return bookings[i].StartsAt.Before(bookings[j].StartsAt)
	})

	// Группируем по статусу
	var confirmed, completed, pending, cancelled []*models.Booking
	var revenue int
	for _, b := range bookings {
		switch b.Status {
		case models.StatusPending:
			pending = append(pending, b)
		case models.StatusConfirmed:
			confirmed = append(confirmed, b)
		case models.StatusCompleted:
			completed = append(completed, b)
			revenue += b.ServicePrice
		case models.StatusCancelledByClient, models.StatusCancelledByMaster:
			cancelled = append(cancelled, b)
		}
	}

	// Сводка
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📅 %s</b>\n\n", formatDate(date)))
	sb.WriteString(fmt.Sprintf("✅ Подтверждено: <b>%d</b>\n", len(confirmed)))
	sb.WriteString(fmt.Sprintf("🏁 Завершено: <b>%d</b>\n", len(completed)))
	sb.WriteString(fmt.Sprintf("⏳ Ожидают подтверждения: <b>%d</b>\n", len(pending)))
	sb.WriteString(fmt.Sprintf("❌ Отменено: <b>%d</b>\n", len(cancelled)))
	if revenue > 0 {
		sb.WriteString(fmt.Sprintf("\n💰 Выручка: <b>%d ₸</b>\n", revenue))
	}

	// Навигация и кнопки разделов — будут в конце
	navRows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("‹ Вчера", fmt.Sprintf("sched_day_%s", date.AddDate(0, 0, -1).Format("2006-01-02"))),
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "sched_today"),
			tgbotapi.NewInlineKeyboardButtonData("Завтра ›", fmt.Sprintf("sched_day_%s", date.AddDate(0, 0, 1).Format("2006-01-02"))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Выбрать день", "sched_select_month"),
		),
	}

	var sectionRows [][]tgbotapi.InlineKeyboardButton
	if len(pending) > 0 {
		sectionRows = append(sectionRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⏳ Ожидают подтверждения (%d)", len(pending)),
				fmt.Sprintf("sched_pending_%s", date.Format("2006-01-02")),
			),
		))
	}
	if len(cancelled) > 0 {
		sectionRows = append(sectionRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("❌ Отменённые (%d)", len(cancelled)),
				fmt.Sprintf("sched_cancelled_%s", date.Format("2006-01-02")),
			),
		))
	}

	bottomKeyboard := tgbotapi.NewInlineKeyboardMarkup(append(sectionRows, navRows...)...)

	if len(bookings) == 0 {
		h.inst.SendWithInlineKeyboard(chatID, fmt.Sprintf("На %s записей нет 🌿", formatDate(date)), bottomKeyboard)
		return
	}

	// 1. Отправляем сводку — без кнопок
	h.inst.SendMessage(chatID, sb.String())

	// 2. Подтверждённые — каждая отдельным сообщением с кнопками
	for i, b := range confirmed {
		endTime := b.StartsAt.Add(time.Duration(b.ServiceDurationMin) * time.Minute)
		text := fmt.Sprintf(
			"✅ <b>%d. %s–%s</b> · %s\n💅 %s\n📱 %s\n💰 %d ₸",
			i+1,
			b.StartsAt.Format("15:04"),
			endTime.Format("15:04"),
			b.ClientName,
			b.ServiceName,
			b.ClientPhone,
			b.ServicePrice,
		)
		btnRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏁 Завершить", fmt.Sprintf("master_complete_%d", b.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", fmt.Sprintf("master_cancel_%d", b.ID)),
		)
		h.inst.SendWithInlineKeyboard(chatID, text, tgbotapi.NewInlineKeyboardMarkup(btnRow))
	}

	// 3. Завершённые — одним списком
	if len(completed) > 0 {
		var compSb strings.Builder
		compSb.WriteString("🏁 <b>Завершённые:</b>\n\n")
		for i, b := range completed {
			compSb.WriteString(fmt.Sprintf(
				"%d. %s · %s\n    💅 %s · %d ₸\n",
				i+1,
				b.StartsAt.Format("15:04"),
				b.ClientName,
				b.ServiceName,
				b.ServicePrice,
			))
		}
		h.inst.SendMessage(chatID, compSb.String())
	}

	// 4. Кнопки отменённых и навигация по датам
	h.inst.SendWithInlineKeyboard(chatID, "🗓 Навигация", bottomKeyboard)
}

// showPendingBookings — экран «Ожидают подтверждения»
func (h *Handler) showPendingBookings(ctx context.Context, chatID int64, date time.Time, bookings []*models.Booking) {
	backBtn := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад к расписанию", fmt.Sprintf("sched_day_%s", date.Format("2006-01-02"))),
		),
	)

	var pending []*models.Booking
	for _, b := range bookings {
		if b.Status == models.StatusPending {
			pending = append(pending, b)
		}
	}

	if len(pending) == 0 {
		h.inst.SendWithInlineKeyboard(chatID, "⏳ Нет записей, ожидающих подтверждения", backBtn)
		return
	}

	h.inst.SendWithInlineKeyboard(chatID, fmt.Sprintf("⏳ <b>Ожидают подтверждения — %s</b>", formatDate(date)), backBtn)

	for _, b := range pending {
		endTime := b.StartsAt.Add(time.Duration(b.ServiceDurationMin) * time.Minute)
		text := fmt.Sprintf(
			"⏳ <b>%s–%s</b> · %s\n💅 %s\n📱 %s\n💰 %d ₸",
			b.StartsAt.Format("15:04"),
			endTime.Format("15:04"),
			b.ClientName,
			b.ServiceName,
			b.ClientPhone,
			b.ServicePrice,
		)
		btnRow := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("admin_confirm_%d", b.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("admin_reject_%d", b.ID)),
		)
		h.inst.SendWithInlineKeyboard(chatID, text, tgbotapi.NewInlineKeyboardMarkup(btnRow))
	}
}

// showCancelledBookings — экран «Отменённые»
func (h *Handler) showCancelledBookings(ctx context.Context, chatID int64, date time.Time, bookings []*models.Booking) {
	backBtn := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад к расписанию", fmt.Sprintf("sched_day_%s", date.Format("2006-01-02"))),
		),
	)

	var cancelled []*models.Booking
	for _, b := range bookings {
		if b.Status == models.StatusCancelledByClient || b.Status == models.StatusCancelledByMaster {
			cancelled = append(cancelled, b)
		}
	}

	if len(cancelled) == 0 {
		h.inst.SendWithInlineKeyboard(chatID, "❌ Отменённых записей нет", backBtn)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("❌ <b>Отменённые — %s</b>\n\n", formatDate(date)))

	for i, b := range cancelled {
		statusLabel := bookingStatusLabel(b.Status)
		sb.WriteString(fmt.Sprintf(
			"%d. %s · %s\n    💅 %s · %d ₸\n    %s\n\n",
			i+1,
			b.StartsAt.Format("15:04"),
			b.ClientName,
			b.ServiceName,
			b.ServicePrice,
			statusLabel,
		))
	}

	h.inst.SendWithInlineKeyboard(chatID, sb.String(), backBtn)
}

func bookingStatusLabel(status string) string {
	switch status {
	case models.StatusPending:
		return "⏳ Ожидает подтверждения"
	case models.StatusConfirmed:
		return "✅ Подтверждена"
	case models.StatusCompleted:
		return "🏁 Завершена"
	case models.StatusCancelledByClient:
		return "❌ Отменена клиентом"
	case models.StatusCancelledByMaster:
		return "❌ Отменена мастером"
	default:
		return "❓ Неизвестный статус"
	}
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
	// if svc.IsActive {
	// 	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
	// 		tgbotapi.NewInlineKeyboardButtonData("🚫 Скрыть", fmt.Sprintf("svc_hide_%d", svcID)),
	// 	))
	// } else {
	// 	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
	// 		tgbotapi.NewInlineKeyboardButtonData("✅ Показать", fmt.Sprintf("svc_show_%d", svcID)),
	// 	))
	// }
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

	// Сохраняем текст в сессию
	session := h.inst.GetSession(userID)
	session.BroadcastText = tmpl
	session.Step = models.StepIdle
	h.inst.SetSession(userID, session)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Отправить", "broadcast_send"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить", "broadcast_edit"),
		),
	)
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
	return fmt.Sprintf("%s, %d %s, %02d:%02d",
		strings.Title(days[t.Weekday()]), t.Day(), months[t.Month()], t.Hour(), t.Minute())
}

func formatDate(t time.Time) string {
	days := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	months := []string{"января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%s, %d %s", days[t.Weekday()], t.Day(), months[t.Month()-1])
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

func (h *Handler) handleNewServiceName(ctx context.Context, msg *tgbotapi.Message) {
	name := strings.TrimSpace(msg.Text)
	if name == "" {
		h.inst.SendMessage(msg.Chat.ID, "Название не может быть пустым. Попробуйте ещё раз:")
		return
	}
	session := h.inst.GetSession(msg.From.ID)
	session.PendingService.Name = name
	session.Step = models.StepAwaitServicePrice
	h.inst.SetSession(msg.From.ID, session)
	h.inst.SendMessage(msg.Chat.ID, "Введите цену в тенге (только число):")
}

func (h *Handler) handleNewServicePrice(ctx context.Context, msg *tgbotapi.Message) {
	price, err := strconv.Atoi(strings.TrimSpace(msg.Text))
	if err != nil || price <= 0 {
		h.inst.SendMessage(msg.Chat.ID, "Введите корректную цену (например: 3000):")
		return
	}
	session := h.inst.GetSession(msg.From.ID)
	session.PendingService.Price = price
	session.Step = models.StepAwaitServiceDuration
	h.inst.SetSession(msg.From.ID, session)
	h.inst.SendMessage(msg.Chat.ID, "Введите длительность в минутах (например: 60):")
}

func (h *Handler) handleNewServiceDuration(ctx context.Context, msg *tgbotapi.Message) {
	dur, err := strconv.Atoi(strings.TrimSpace(msg.Text))
	if err != nil || dur <= 0 {
		h.inst.SendMessage(msg.Chat.ID, "Введите корректную длительность в минутах:")
		return
	}
	session := h.inst.GetSession(msg.From.ID)
	session.PendingService.DurationMin = dur
	p := session.PendingService

	svc := &models.Service{
		MasterID:    h.inst.Master.ID,
		Name:        p.Name,
		Price:       p.Price,
		DurationMin: dur,
		IsActive:    true,
	}
	_, err = h.repos.Service.Create(ctx, svc)
	if err != nil {
		h.inst.SendMessage(msg.Chat.ID, "Ошибка при сохранении. Попробуйте снова.")
		return
	}

	session.Step = models.StepIdle
	session.PendingService = models.PendingService{}
	h.inst.SetSession(msg.From.ID, session)

	h.inst.SendMessage(msg.Chat.ID, fmt.Sprintf(
		"✅ Услуга добавлена!\n\n💅 %s\n💰 %d ₸\n⏱ %s",
		p.Name, p.Price, formatDuration(dur),
	))
	h.handleServices(ctx, msg.Chat.ID)
}
func (h *Handler) showServiceEditMenu(ctx context.Context, chatID int64, svcID int) {
	svc, err := h.repos.Service.GetByID(ctx, svcID)
	if err != nil {
		return
	}
	priceStr := fmt.Sprintf("%d ₸", svc.Price)
	if svc.PriceFrom {
		priceStr = "от " + priceStr
	}
	text := fmt.Sprintf("<b>%s</b>\n%s — %s\n\nЧто изменить?",
		svc.Name, priceStr, formatDuration(svc.DurationMin))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Название", fmt.Sprintf("svc_edit_name_%d", svcID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Стоимость", fmt.Sprintf("svc_edit_price_%d", svcID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏱ Продолжительность", fmt.Sprintf("svc_edit_duration_%d", svcID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← Назад", fmt.Sprintf("admin_svc_%d", svcID)),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, text, keyboard)
}

func (h *Handler) handleEditServiceName(ctx context.Context, msg *tgbotapi.Message) {
	name := strings.TrimSpace(msg.Text)
	if name == "" {
		h.inst.SendMessage(msg.Chat.ID, "Название не может быть пустым:")
		return
	}
	session := h.inst.GetSession(msg.From.ID)
	h.repos.Service.UpdateName(ctx, session.ServiceID, name)
	h.finishEdit(ctx, msg.Chat.ID, msg.From.ID, session.ServiceID)
}

func (h *Handler) handleEditServicePrice(ctx context.Context, msg *tgbotapi.Message) {
	input := strings.TrimSpace(strings.ToLower(msg.Text))
	priceFrom := strings.HasPrefix(input, "от ")
	numStr := strings.TrimPrefix(input, "от ")
	price, err := strconv.Atoi(strings.TrimSpace(numStr))
	if err != nil || price <= 0 {
		h.inst.SendMessage(msg.Chat.ID, "Некорректная сумма. Введите число, например: 3000")
		return
	}
	session := h.inst.GetSession(msg.From.ID)
	h.repos.Service.UpdatePrice(ctx, session.ServiceID, price, priceFrom)
	h.finishEdit(ctx, msg.Chat.ID, msg.From.ID, session.ServiceID)
}

func (h *Handler) handleEditServiceDuration(ctx context.Context, msg *tgbotapi.Message) {
	dur, err := strconv.Atoi(strings.TrimSpace(msg.Text))
	if err != nil || dur <= 0 {
		h.inst.SendMessage(msg.Chat.ID, "Введите длительность в минутах, например: 60")
		return
	}
	session := h.inst.GetSession(msg.From.ID)
	h.repos.Service.UpdateDuration(ctx, session.ServiceID, dur)
	h.finishEdit(ctx, msg.Chat.ID, msg.From.ID, session.ServiceID)
}

func (h *Handler) finishEdit(ctx context.Context, chatID int64, userID int64, svcID int) {
	h.inst.ClearSession(userID)
	h.inst.SendMessage(chatID, "✅ Сохранено")
	h.showServiceActions(ctx, chatID, svcID)
}

func (h *Handler) handleBlockList(ctx context.Context, chatID int64) {
	slots, err := h.repos.BlockedSlot.GetUpcoming(ctx, h.inst.Master.ID)
	if err != nil || len(slots) == 0 {
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "work_schedule"),
			),
		)
		h.inst.SendWithInlineKeyboard(chatID, "Активных блокировок нет ✅", kb)
		return
	}

	text := "🚫 <b>Активные блокировки:</b>\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range slots {
		// Весь день
		if s.StartsAt.Hour() == 0 && s.EndsAt.Sub(s.StartsAt).Hours() == 24 {
			text += fmt.Sprintf("📅 %s — весь день\n", s.StartsAt.Format("02.01.2006"))
		} else {
			text += fmt.Sprintf("⏰ %s %s–%s\n",
				s.StartsAt.Format("02.01"),
				s.StartsAt.Format("15:04"),
				s.EndsAt.Format("15:04"))
		}
		if s.Reason != "" {
			text += fmt.Sprintf("   📝 %s\n", s.Reason)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🔓 Снять %s", s.StartsAt.Format("02.01")),
				fmt.Sprintf("block_delete_%d", s.ID),
			),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "work_schedule"),
	))

	h.inst.SendWithInlineKeyboard(chatID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) handleBlockDelete(ctx context.Context, chatID int64, id int) {
	err := h.repos.BlockedSlot.DeleteByID(ctx, id)
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка при снятии блокировки.")
		return
	}
	h.inst.SendMessage(chatID, "Блокировка снята ✅")
	h.handleBlockList(ctx, chatID)
}
