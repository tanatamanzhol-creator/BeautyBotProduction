package repository

import (
	"context"
	"time"

	"beauty-bot/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BlockedSlotRepo struct {
	db *pgxpool.Pool
}

func NewBlockedSlotRepo(db *pgxpool.Pool) *BlockedSlotRepo {
	return &BlockedSlotRepo{db: db}
}

func (r *BlockedSlotRepo) Create(ctx context.Context, masterID int, startsAt, endsAt time.Time, reason string) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		INSERT INTO blocked_slots (master_id, starts_at, ends_at, reason)
		VALUES ($1, $2, $3, $4)
	`, masterID, startsAt, endsAt, reason)
	return err
}

func (r *BlockedSlotRepo) DeleteForDay(ctx context.Context, masterID int, date time.Time) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	_, err := r.db.Exec(ctx, `
		DELETE FROM blocked_slots
		WHERE master_id=$1 AND starts_at >= $2 AND ends_at <= $3
	`, masterID, dayStart, dayEnd)
	return err
}
