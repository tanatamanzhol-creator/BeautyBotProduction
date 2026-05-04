package admin_bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"beauty-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handlePrepaymentMenu(ctx context.Context, chatID int64) {
	master := h.inst.Master

	status := "❌ Выключена"
	if master.PrepaymentEnabled {
		status = "✅ Включена"
	}

	amount := "не указана"
	if master.PrepaymentAmount > 0 {
		amount = fmt.Sprintf("%d ₸", master.PrepaymentAmount)
	}

	details := "не указаны"
	if master.PrepaymentDetails != "" {
		details = master.PrepaymentDetails
	}

	toggleLabel := "Включить предоплату"
	if master.PrepaymentEnabled {
		toggleLabel = "Выключить предоплату"
	}

	text := fmt.Sprintf(
		"💳 <b>Предоплата</b>\n\n"+
			"Статус: %s\n"+
			"Сумма: %s\n"+
			"Реквизиты: %s",
		status, amount, details,
	)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(toggleLabel, "prepayment_toggle"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Изменить сумму", "prepayment_set_amount"),
			tgbotapi.NewInlineKeyboardButtonData("🏦 Изменить реквизиты", "prepayment_set_details"),
		),
	)

	h.inst.SendWithInlineKeyboard(chatID, text, kb)
}

func (h *Handler) handlePrepaymentAmount(ctx context.Context, msg *tgbotapi.Message, session *models.SessionState) {
	text := strings.TrimSpace(msg.Text)
	amount, err := strconv.Atoi(text)
	if err != nil || amount <= 0 {
		h.inst.SendMessage(msg.Chat.ID, "Введите корректную сумму (только число, например 500):")
		return
	}

	master := h.inst.Master
	if err := h.repos.Master.UpdatePrepayment(ctx, master.ID, master.PrepaymentEnabled, amount, master.PrepaymentDetails); err != nil {
		h.inst.SendMessage(msg.Chat.ID, "Ошибка при сохранении. Попробуйте ещё раз.")
		return
	}
	master.PrepaymentAmount = amount

	session.Step = ""
	h.inst.SetSession(msg.From.ID, session)

	h.inst.SendMessage(msg.Chat.ID, fmt.Sprintf("✅ Сумма предоплаты установлена: %d ₸", amount))
	h.handlePrepaymentMenu(ctx, msg.Chat.ID)
}

func (h *Handler) handlePrepaymentDetails(ctx context.Context, msg *tgbotapi.Message, session *models.SessionState) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		h.inst.SendMessage(msg.Chat.ID, "Реквизиты не могут быть пустыми. Введите ещё раз:")
		return
	}

	master := h.inst.Master
	if err := h.repos.Master.UpdatePrepayment(ctx, master.ID, master.PrepaymentEnabled, master.PrepaymentAmount, text); err != nil {
		h.inst.SendMessage(msg.Chat.ID, "Ошибка при сохранении. Попробуйте ещё раз.")
		return
	}
	master.PrepaymentDetails = text

	session.Step = ""
	h.inst.SetSession(msg.From.ID, session)

	h.inst.SendMessage(msg.Chat.ID, "✅ Реквизиты сохранены.")
	h.handlePrepaymentMenu(ctx, msg.Chat.ID)
}
