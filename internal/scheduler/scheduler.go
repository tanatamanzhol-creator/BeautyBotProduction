package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"beauty-bot/internal/bot"
	"beauty-bot/internal/types"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	manager *bot.Manager
	repos   *types.Repos
	cron    *cron.Cron
}

func New(manager *bot.Manager, repos *types.Repos) *Scheduler {
	return &Scheduler{
		manager: manager,
		repos:   repos,
		cron:    cron.New(),
	}
}

func (s *Scheduler) Start() {
	// Auto-confirm pending bookings older than 30 min
	s.cron.AddFunc("*/5 * * * *", func() { s.autoConfirmPending() })

	// Send 24h reminders
	s.cron.AddFunc("*/10 * * * *", func() { s.sendReminders24h() })

	// Send 2h reminders
	s.cron.AddFunc("*/10 * * * *", func() { s.sendReminders2h() })

	// Send review requests (3h after booking ends)
	s.cron.AddFunc("*/10 * * * *", func() { s.sendReviewRequests() })

	// Daily schedule to masters at 8:00
	s.cron.AddFunc("0 8 * * *", func() { s.sendDailySchedule() })

	s.cron.Start()
	log.Println("Scheduler started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) autoConfirmPending() {
	ctx := context.Background()
	before := time.Now().Add(-30 * time.Minute)

	bookings, err := s.repos.Booking.GetPendingForAutoConfirm(ctx, before)
	if err != nil {
		return
	}

	for _, b := range bookings {
		if err := s.repos.Booking.Confirm(ctx, b.ID, "auto"); err != nil {
			continue
		}

		// Notify client via client bot
		s.manager.NotifyClientConfirmed(b.MasterID, b)

		// Notify master via admin bot
		master, err := s.repos.Master.GetByID(ctx, b.MasterID)
		if err != nil {
			continue
		}
		adminInst := s.manager.GetAdminBot(b.MasterID)
		if adminInst != nil {
			adminInst.SendMessage(master.MasterTelegramID,
				fmt.Sprintf("✅ Запись автоподтверждена:\n%s — %s %s",
					b.ClientName, formatDate(b.StartsAt), b.StartsAt.Format("15:04")))
		}

		log.Printf("Auto-confirmed booking %d", b.ID)
	}
}

func (s *Scheduler) sendReminders24h() {
	ctx := context.Background()
	bookings, err := s.repos.Booking.GetNeedingReminder24h(ctx)
	if err != nil {
		return
	}

	for _, b := range bookings {
		if isQuietHours() {
			continue
		}

		inst := s.manager.GetClientBot(b.MasterID)
		if inst == nil {
			continue
		}

		master, _ := s.repos.Master.GetByID(ctx, b.MasterID)
		addr := ""
		if master != nil && master.Address != "" {
			addr = "\n📍 " + master.Address
		}

		text := fmt.Sprintf(
			"Напоминаем о вашей записи завтра 🔔\n\n💅 %s\n📅 %s — %s%s\n\nЖдём вас! 🤍",
			b.ServiceName, formatDate(b.StartsAt), b.StartsAt.Format("15:04"), addr,
		)
		inst.SendMessage(b.ClientTelegramID, text)
		s.repos.Booking.MarkReminder24hSent(ctx, b.ID)
		log.Printf("Sent 24h reminder for booking %d", b.ID)
	}
}

func (s *Scheduler) sendReminders2h() {
	ctx := context.Background()
	bookings, err := s.repos.Booking.GetNeedingReminder2h(ctx)
	if err != nil {
		return
	}

	for _, b := range bookings {
		inst := s.manager.GetClientBot(b.MasterID)
		if inst == nil {
			continue
		}

		master, _ := s.repos.Master.GetByID(ctx, b.MasterID)
		addr := ""
		if master != nil && master.Address != "" {
			addr = "\n📍 " + master.Address
		}

		text := fmt.Sprintf(
			"Совсем скоро ваша запись ⏰\n\n💅 %s\n📅 Сегодня в %s%s\n\nДо встречи! 🤍",
			b.ServiceName, b.StartsAt.Format("15:04"), addr,
		)
		inst.SendMessage(b.ClientTelegramID, text)
		s.repos.Booking.MarkReminder2hSent(ctx, b.ID)
		log.Printf("Sent 2h reminder for booking %d", b.ID)
	}
}

func (s *Scheduler) sendReviewRequests() {
	ctx := context.Background()
	bookings, err := s.repos.Booking.GetNeedingReviewRequest(ctx)
	if err != nil {
		return
	}

	for _, b := range bookings {
		if isQuietHours() {
			continue
		}

		inst := s.manager.GetClientBot(b.MasterID)
		if inst == nil {
			continue
		}

		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✏️ Оставить отзыв", "leave_review"),
				tgbotapi.NewInlineKeyboardButtonData("Пропустить", "back_to_menu"),
			),
		)
		inst.SendWithInlineKeyboard(b.ClientTelegramID,
			"Как прошёл ваш визит? 🤍\n\nНапишите пару слов — это очень важно для мастера!", kb)

		// Store booking ID in session so review is linked
		session := inst.GetSession(b.ClientTelegramID)
		session.BookingID = b.ID
		inst.SetSession(b.ClientTelegramID, session)

		s.repos.Booking.MarkReviewRequested(ctx, b.ID)

		log.Printf("Sent review request for booking %d", b.ID)
	}

}

func (s *Scheduler) sendDailySchedule() {
	ctx := context.Background()
	allBots := s.manager.GetAllClientBots()

	for _, clientInst := range allBots {
		master, err := s.repos.Master.GetByID(ctx, clientInst.Master.ID)
		if err != nil || master.MasterTelegramID == 0 {
			continue
		}

		bookings, err := s.repos.Booking.GetActiveForDay(ctx, master.ID, time.Now())
		if err != nil || len(bookings) == 0 {
			continue
		}

		text := "📋 <b>Ваши записи на сегодня:</b>\n\n"
		total := 0
		for _, b := range bookings {
			text += fmt.Sprintf("⏰ <b>%s</b> — %s\n   💅 %s\n\n",
				b.StartsAt.Format("15:04"), b.ClientName, b.ServiceName)
			total += b.ServicePrice
		}
		text += fmt.Sprintf("Всего: <b>%d клиентов</b> / ~%d ₸ 💪", len(bookings), total)

		adminInst := s.manager.GetAdminBot(master.ID)
		if adminInst != nil {
			adminInst.SendMessage(master.MasterTelegramID, text)
		}
	}
}

func isQuietHours() bool {
	h := time.Now().Hour()
	return h < 9 || h >= 21
}

func formatDate(t time.Time) string {
	months := []string{"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря"}
	return fmt.Sprintf("%d %s", t.Day(), months[t.Month()])
}
