package repository

import (
	"context"
	"database/sql"
	"beauty-bot/internal/models"

	"beauty-bot/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewRepo struct {
	db *pgxpool.Pool
}

func NewReviewRepo(db *pgxpool.Pool) *ReviewRepo {
	return &ReviewRepo{db: db}
}

func (r *ReviewRepo) Create(ctx context.Context, masterID, clientID int, bookingID *int, text string) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
    INSERT INTO reviews (master_id, client_id, booking_id, text)
    VALUES ($1,$2,$3,$4)
`, masterID, clientID, bookingID, text)
	return err
}

func (r *ReviewRepo) GetAllForMaster(ctx context.Context, masterID int) ([]*models.Review, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT 
	r.id, r.master_id, r.client_id, r.booking_id, 
	r.text, r.created_at,
	COALESCE(c.name, ''),
	COALESCE(s.name, '')
		FROM reviews r
		LEFT JOIN clients c ON c.id = r.client_id
LEFT JOIN bookings b ON b.id = r.booking_id
LEFT JOIN services s ON s.id = b.service_id
		WHERE r.master_id=$1
		ORDER BY r.created_at DESC
	`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*models.Review
	for rows.Next() {
	var bookingID sql.NullInt64

	rev := &models.Review{}
	if err := rows.Scan(
		&rev.ID,
		&rev.MasterID,
		&rev.ClientID,
		&bookingID,
		&rev.Text,
		&rev.CreatedAt,
		&rev.ClientName,
		&rev.ServiceName,
	); err != nil {
		return nil, err
	}

	// корректно обрабатываем NULL
	if bookingID.Valid {
		rev.BookingID = int(bookingID.Int64)
	} else {
		rev.BookingID = 0 // или можешь оставить как есть, если 0 по умолчанию ок
	}

	reviews = append(reviews, rev)
}

// 👇 обязательно добавь это после цикла
if err := rows.Err(); err != nil {
	return nil, err
}
	return reviews, nil
}
