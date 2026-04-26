package client_bot

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	userID := msg.From.ID

	client, err := h.repos.Client.GetOrCreate(ctx,
		h.inst.Master.ID, userID, msg.From.UserName)
	if err != nil {
		log.Printf("GetOrCreate client error: %v", err)
		return
	}

	// Проверка на нажатие "Главное меню"
	if msg.Text == "🏠 Главное меню" {
		h.inst.ClearSession(userID) // сброс всех шагов сессии
		h.sendMainMenu(ctx, msg.Chat.ID, "Вы вернулись в главное меню 👇")
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			h.handleStart(ctx, msg, client)
		case "privacy":
			h.handlePrivacy(ctx, msg)
		}
		return
	}

	if !client.ConsentGiven {
		h.sendConsentScreen(ctx, msg.Chat.ID)
		return
	}

	session := h.inst.GetSession(userID)

	switch session.Step {
	case models.StepAwaitName:
		h.handleNameInput(ctx, msg, client, session)
	case models.StepAwaitPhone:
		h.handlePhoneInput(ctx, msg, client, session)
	case models.StepAwaitReview:
		h.handleReviewInput(ctx, msg, client, session)
	default:
		switch msg.Text {
		case "📅 Записаться":
			h.handleBookingStart(ctx, msg, client)
		case "📋 Мои записи":
			h.handleMyBookings(ctx, msg, client)
		case "💬 Вопрос":
			h.handleQuestion(ctx, msg)
		case "🗺 Адрес":
			h.handleAddress(ctx, msg.Chat.ID)
		default:
			h.sendMainMenu(ctx, msg.Chat.ID, "Выберите действие из меню 👇")
		}
	}
}

func (h *Handler) handleStart(ctx context.Context, msg *tgbotapi.Message, client *models.Client) {
	h.inst.ClearSession(msg.From.ID)
	if client.ConsentGiven {
		name := client.Name
		if name == "" {
			name = msg.From.FirstName
		}
		h.sendMainMenu(ctx, msg.Chat.ID, "С возвращением, <b>"+name+"</b>! 👋\nЧем могу помочь?")
		return
	}
	h.sendConsentScreen(ctx, msg.Chat.ID)
}

func (h *Handler) sendConsentScreen(ctx context.Context, chatID int64) {
	master := h.inst.Master
	welcome := master.WelcomeText
	if welcome == "" {
		welcome = "Привет! 👋 Я бот мастера <b>" + master.Name + "</b>\n\nЗдесь вы можете записаться онлайн быстро и удобно — без звонков и ожидания."
	}
	text := welcome + "\n\nДля работы я собираю:\n• Ваше имя\n• Номер телефона\n\nДанные используются только для связи с вами и не передаются третьим лицам.\n\n 📌 Пожалуйста, нажмите кнопку ниже, чтобы согласиться и продолжить запись."
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Принимаю и продолжаю", "consent_accept"),
		),
	)
	h.inst.SendWithInlineKeyboard(chatID, text, kb)
}

func (h *Handler) sendMainMenu(ctx context.Context, chatID int64, text string) {
	if text == "" {
		return
	}
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 Записаться"),
			tgbotapi.NewKeyboardButton("📋 Мои записи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💬 Вопрос"),
			tgbotapi.NewKeyboardButton("🗺 Адрес"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню (начать)"), // добавляем кнопку для главного меню
		),
	)
	kb.ResizeKeyboard = true
	h.inst.SendWithReplyKeyboard(chatID, text, kb)
}

func (h *Handler) handlePrivacy(ctx context.Context, msg *tgbotapi.Message) {
	h.inst.SendMessage(msg.Chat.ID,
		"🔒 <b>Политика конфиденциальности</b>\n\n"+
			"Собираем: имя и номер телефона.\n"+
			"Используем только для связи с вами.\n"+
			"Не передаём третьим лицам.\n\n"+
			"Чтобы удалить данные — напишите мастеру.")
}

func (h *Handler) handleQuestion(ctx context.Context, msg *tgbotapi.Message) {
	master := h.inst.Master

	var telegramContact string
	if master.MasterTelegramID != 0 {
		telegramContact = fmt.Sprintf("tg://user?id=%d", master.MasterTelegramID)
	} else {
		telegramContact = "К сожалению, мастер еще не добавил свой Telegram"
	}

	text := fmt.Sprintf(
		"💬 Напишите сообщение напрямую мастеру в "+
			"Telegram: %s\nАдрес: %s",
		telegramContact, master.Address,
	)

	h.inst.SendMessage(msg.Chat.ID, text)
}

func (h *Handler) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	h.inst.API.Send(tgbotapi.NewCallback(cb.ID, ""))

	userID := cb.From.ID
	chatID := cb.Message.Chat.ID
	data := cb.Data

	client, err := h.repos.Client.GetOrCreate(ctx,
		h.inst.Master.ID, userID, cb.From.UserName)
	if err != nil {
		return
	}

	switch {
	case data == "consent_accept":
		h.repos.Client.SaveConsent(ctx, client.ID)
		h.sendMainMenu(ctx, chatID, "Отлично! Выберите что вас интересует 👇")

	case data == "booking_start":
		h.handleBookingStartCallback(ctx, chatID, client)

	case strings.HasPrefix(data, "cat_"):
		h.handleCategorySelected(ctx, chatID, userID, data)

	case strings.HasPrefix(data, "svc_"):
		h.handleServiceSelected(ctx, chatID, userID, data)

	case strings.HasPrefix(data, "date_"):
		h.handleDateSelected(ctx, chatID, userID, data)

	case strings.HasPrefix(data, "time_"):
		h.handleTimeSelected(ctx, chatID, userID, data)

	case data == "confirm_booking":
		h.handleConfirmBooking(ctx, chatID, userID, client)

	case data == "back_to_menu":
		h.inst.ClearSession(userID)
		h.sendMainMenu(ctx, chatID, "Выберите действие 👇")

	case data == "back_to_services":
		h.handleBookingStartCallback(ctx, chatID, client)

	case strings.HasPrefix(data, "cancel_booking_"):
		h.handleCancelBookingPrompt(ctx, chatID, userID, data)

	case strings.HasPrefix(data, "cancel_reason_"):
		h.handleCancelReason(ctx, chatID, userID, data)

	case strings.HasPrefix(data, "reschedule_"):
		h.handleReschedule(ctx, chatID, userID, data, client)

	case data == "no_broadcast":
		h.repos.Client.SetNoBroadcast(ctx, client.ID, true)
		h.inst.SendMessage(chatID, "Вы отписались от рассылок 🔕")

	case data == "leave_review":
		session := h.inst.GetSession(userID)
		session.Step = models.StepAwaitReview
		h.inst.SetSession(userID, session)
		h.inst.SendMessage(chatID, "Напишите ваш отзыв ✏️\n\nМастер обязательно прочитает его 🤍")
	}
}

func (h *Handler) handleAddress(ctx context.Context, chatID int64) {
	master, err := h.repos.Master.GetByID(ctx, h.inst.Master.ID)
	if err != nil {
		h.inst.SendMessage(chatID, "Ошибка получения адреса.")
		return
	}

	var url string

	if master.PoiID != "" {
		url = fmt.Sprintf("https://2gis.kz/pavlodar/geo/%s", master.PoiID)
	} else if master.Latitude != 0 && master.Longitude != 0 {
		// правильный порядок: latitude, longitude
		url = fmt.Sprintf("https://2gis.kz/geo/%f,%f", master.Latitude, master.Longitude)
	} else {
		h.inst.SendMessage(chatID, "Адрес не указан.")
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🗺 Открыть в 2ГИС", url),
		),
	)
	text := fmt.Sprintf("📍 Моё местоположение:\n%s", master.Address)

	h.inst.SendWithInlineKeyboard(chatID, text, keyboard)
}
