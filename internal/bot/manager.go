package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"beauty-bot/internal/bot/admin_bot"
	"beauty-bot/internal/bot/client_bot"
	"beauty-bot/internal/models"
	"beauty-bot/internal/types"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Manager struct {
	repos      *types.Repos
	clientBots map[int]*types.BotInstance
	adminBots  map[int]*types.BotInstance
	mu         sync.RWMutex
	adminTgID  int64
}

func NewManager(repos *types.Repos, adminTgID int64) *Manager {
	return &Manager{
		repos:      repos,
		clientBots: make(map[int]*types.BotInstance),
		adminBots:  make(map[int]*types.BotInstance),
		adminTgID:  adminTgID,
	}
}

func (m *Manager) StartAll(ctx context.Context) error {
	masters, err := m.repos.Master.GetAll(ctx)
	if err != nil {
		return err
	}
	for _, master := range masters {
		if err := m.StartMaster(ctx, master); err != nil {
			log.Printf("Failed to start bots for master %d (%s): %v", master.ID, master.Name, err)
		}
	}
	return nil
}

func (m *Manager) StartMaster(ctx context.Context, master *models.Master) error {
	clientAPI, err := tgbotapi.NewBotAPI(master.ClientBotToken)
	if err != nil {
		return fmt.Errorf("client bot: %w", err)
	}
	adminAPI, err := tgbotapi.NewBotAPI(master.AdminBotToken)
	if err != nil {
		return fmt.Errorf("admin bot: %w", err)
	}

	clientInst := &types.BotInstance{API: clientAPI, Master: master, IsAdmin: false, Notifier: m}
	adminInst := &types.BotInstance{API: adminAPI, Master: master, IsAdmin: true, Notifier: m}

	m.mu.Lock()
	m.clientBots[master.ID] = clientInst
	m.adminBots[master.ID] = adminInst
	m.mu.Unlock()

	go m.runClientBot(ctx, clientInst)
	go m.runAdminBot(ctx, adminInst)

	log.Printf("Started bots for master: %s (ID: %d)", master.Name, master.ID)
	return nil
}

func (m *Manager) runClientBot(ctx context.Context, inst *types.BotInstance) {
	handler := client_bot.NewHandler(inst, m.repos)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := inst.API.GetUpdatesChan(u)
	log.Printf("[ClientBot] %s started", inst.Master.Name)
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			go safeHandle(ctx, inst, func() {
				handler.Handle(ctx, update)
			})
		}
	}
}

func (m *Manager) runAdminBot(ctx context.Context, inst *types.BotInstance) {
	handler := admin_bot.NewHandler(inst, m.repos)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := inst.API.GetUpdatesChan(u)
	log.Printf("[AdminBot] %s started", inst.Master.Name)
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			go safeHandle(ctx, inst, func() {
				handler.Handle(ctx, update)
			})
		}
	}
}

// safeHandle wraps a handler call with panic recovery
// so one user's panic doesn't crash the whole bot
func safeHandle(ctx context.Context, inst *types.BotInstance, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC RECOVERED] master=%s error=%v", inst.Master.Name, r)
		}
	}()
	fn()
}

func (m *Manager) GetClientBot(masterID int) *types.BotInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clientBots[masterID]
}

func (m *Manager) GetAdminBot(masterID int) *types.BotInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.adminBots[masterID]
}

func (m *Manager) GetAllClientBots() []*types.BotInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var bots []*types.BotInstance
	for _, b := range m.clientBots {
		bots = append(bots, b)
	}
	return bots
}

// ── types.Notifier implementation ────────────────────────────────────────

func (m *Manager) NotifyMasterNewBooking(masterID int, masterTelegramID int64, booking *models.Booking) {
	inst := m.GetAdminBot(masterID)
	if inst == nil {
		return
	}
	text := fmt.Sprintf(
		"🔔 <b>Новая заявка!</b>\n\n👤 %s\n📱 %s\n💅 %s\n📅 %s — %s",
		booking.ClientName, booking.ClientPhone,
		booking.ServiceName,
		formatDate(booking.StartsAt), booking.StartsAt.Format("15:04"),
	)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Принять", fmt.Sprintf("admin_confirm_%d", booking.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("admin_reject_%d", booking.ID)),
		),
	)
	inst.SendWithInlineKeyboard(masterTelegramID, text, kb)
}

func (m *Manager) NotifyClientConfirmed(masterID int, booking *models.Booking) {
	inst := m.GetClientBot(masterID)
	if inst == nil {
		return
	}

	ctx := context.Background()

	master, _ := m.repos.Master.GetByID(ctx, masterID)

	addr := ""
	if master != nil && master.Address != "" {
		addr = "\n📍 " + master.Address
	}

	// ⏱ время до записи
	durationUntil := time.Until(booking.StartsAt)

	reminderText := ""

	if durationUntil > 24*time.Hour {
		reminderText = "\n\n⏰ Напомним за 24 часа 🔔"
	} else if durationUntil > 2*time.Hour {
		reminderText = "\n\n⏰ Напомним за 2 часа 🔔"
	} else {
		reminderText = "\n\n⏰ Напомним перед записью 🔔"
	}

	text := fmt.Sprintf(
		"Запись подтверждена! ✅\n\n💅 %s\n📅 %s — %s%s%s\n\nЖдём вас!",
		booking.ServiceName,
		formatDate(booking.StartsAt),
		booking.StartsAt.Format("15:04"),
		addr,
		reminderText,
	)

	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📅 Записаться"),
			tgbotapi.NewKeyboardButton("📋 Мои записи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💬 Вопрос"),
			tgbotapi.NewKeyboardButton("🗺 Адрес"),
		),
	)
	inst.SendWithReplyKeyboard(booking.ClientTelegramID, text, kb)
}

func (m *Manager) NotifyClientRejected(masterID int, booking *models.Booking, reason string) {
	inst := m.GetClientBot(masterID)
	if inst == nil {
		return
	}
	text := fmt.Sprintf("К сожалению, мастер не может принять вас в это время %d 😔", booking.StartsAt)
	if reason != "" {
		text += "\n\n" + reason
	}
	text += "\n\nХотите выбрать другое время?"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Другое время", "booking_start"),
			tgbotapi.NewInlineKeyboardButtonData("← Меню", "back_to_menu"),
		),
	)
	inst.SendWithInlineKeyboard(booking.ClientTelegramID, text, kb)
}

func (m *Manager) NotifyClientCancelledByMaster(masterID int, booking *models.Booking) {
	inst := m.GetClientBot(masterID)
	if inst == nil {
		return
	}
	text := fmt.Sprintf(
		"К сожалению, мастер вынуждена отменить вашу запись 😔\n\n💅 %s\n📅 %s — %s\n\nПриносим извинения 🙏",
		booking.ServiceName, formatDate(booking.StartsAt), booking.StartsAt.Format("15:04"),
	)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Записаться снова", "booking_start"),
		),
	)
	inst.SendWithInlineKeyboard(booking.ClientTelegramID, text, kb)
}

func (m *Manager) NotifyMasterClientCancelled(masterID int, masterTelegramID int64, booking *models.Booking, reason string) {
	inst := m.GetAdminBot(masterID)
	if inst == nil {
		return
	}
	text := fmt.Sprintf(
		"❌ Клиент отменил запись\n\n👤 %s\n💅 %s\n📅 %s — %s\n💬 Причина: %s\n\nСлот %s освобождён",
		booking.ClientName, booking.ServiceName,
		formatDate(booking.StartsAt), booking.StartsAt.Format("15:04"),
		reason, booking.StartsAt.Format("15:04"),
	)
	inst.SendMessage(masterTelegramID, text)
}

func (m *Manager) NotifyMasterNewReview(masterID int, masterTelegramID int64, clientName, serviceName, reviewText string) {
	inst := m.GetAdminBot(masterID)
	if inst == nil {
		return
	}
	text := fmt.Sprintf(
		"⭐ <b>Новый отзыв!</b>\n\n👤 %s\n💅 %s\n\n✏️ <i>%s</i>",
		clientName, serviceName, reviewText,
	)
	inst.SendMessage(masterTelegramID, text)
}

func formatDate(t time.Time) string {
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%d %s", t.Day(), months[t.Month()])
}
