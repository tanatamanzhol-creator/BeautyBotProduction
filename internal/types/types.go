package types

import (
	"log"
	"sync"

	"beauty-bot/internal/models"
	"beauty-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Repos holds all repositories — passed everywhere
type Repos struct {
	Master      *repository.MasterRepo
	Client      *repository.ClientRepo
	Service     *repository.ServiceRepo
	Booking     *repository.BookingRepo
	Review      *repository.ReviewRepo
	BlockedSlot *repository.BlockedSlotRepo
}

// BotInstance holds a running bot and its master info
type BotInstance struct {
	API      *tgbotapi.BotAPI
	Master   *models.Master
	IsAdmin  bool
	Sessions sync.Map // map[int64]*models.SessionState

	// Notifier lets client_bot/admin_bot send cross-bot messages
	// Set by manager after both bots are created
	Notifier Notifier
}

// Notifier interface — manager implements this, injected into handlers
type Notifier interface {
	NotifyMasterNewBooking(masterID int, masterTelegramID int64, booking *models.Booking)
	NotifyClientConfirmed(masterID int, booking *models.Booking)
	NotifyClientRejected(masterID int, booking *models.Booking, reason string)
	NotifyClientCancelledByMaster(masterID int, booking *models.Booking)
	NotifyMasterClientCancelled(masterID int, masterTelegramID int64, booking *models.Booking, reason string)
	NotifyMasterNewReview(masterID int, masterTelegramID int64, clientName, serviceName, text string)
}

// Session helpers

func (inst *BotInstance) GetSession(userID int64) *models.SessionState {
	val, _ := inst.Sessions.LoadOrStore(userID, &models.SessionState{})
	return val.(*models.SessionState)
}

func (inst *BotInstance) SetSession(userID int64, state *models.SessionState) {
	inst.Sessions.Store(userID, state)
}

func (inst *BotInstance) ClearSession(userID int64) {
	inst.Sessions.Store(userID, &models.SessionState{})
}

// Send helpers

func (inst *BotInstance) SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	inst.API.Send(msg)
}

func (inst *BotInstance) SendWithInlineKeyboard(chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	// 🔥 важно: не отправляем пустую клавиатуру
	if len(kb.InlineKeyboard) > 0 {
		msg.ReplyMarkup = kb
	}

	_, err := inst.API.Send(msg)
	if err != nil {
		log.Printf("SendWithInlineKeyboard error: %v", err)
	}
}

func (inst *BotInstance) SendWithReplyKeyboard(chatID int64, text string, keyboard tgbotapi.ReplyKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	inst.API.Send(msg)
}

func (inst *BotInstance) RemoveKeyboard(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "⏳")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	inst.API.Send(msg)
}
