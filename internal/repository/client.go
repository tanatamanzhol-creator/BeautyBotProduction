package repository

import (
	"beauty-bot/internal/models"
	"context"
	"time"

	"beauty-bot/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ClientRepo struct {
	db *pgxpool.Pool
}

func NewClientRepo(db *pgxpool.Pool) *ClientRepo {
	return &ClientRepo{db: db}
}

func (r *ClientRepo) GetOrCreate(ctx context.Context, masterID int, telegramID int64, username string) (*models.Client, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	client := &models.Client{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO clients (master_id, telegram_id, telegram_username)
		VALUES ($1, $2, $3)
		ON CONFLICT (master_id, telegram_id) DO UPDATE
			SET telegram_username = EXCLUDED.telegram_username
		RETURNING id, master_id, telegram_id, telegram_username,
		          COALESCE(name,''), COALESCE(phone,''),
		          consent_given, consent_given_at,
		          no_broadcast, is_blocked, created_at, visit_count, last_visit_at
	`, masterID, telegramID, username).Scan(
		&client.ID, &client.MasterID, &client.TelegramID, &client.TelegramUsername,
		&client.Name, &client.Phone, &client.ConsentGiven, &client.ConsentGivenAt,
		&client.NoBroadcast, &client.IsBlocked, &client.CreatedAt, &client.VisitCount, &client.LastVisitAt,
	)
	return client, err
}

func (r *ClientRepo) GetByTelegramID(ctx context.Context, masterID int, telegramID int64) (*models.Client, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	client := &models.Client{}
	err := r.db.QueryRow(ctx, `
		SELECT id, master_id, telegram_id, telegram_username,
		       COALESCE(name,''), COALESCE(phone,''),
		       consent_given, consent_given_at,
		       no_broadcast, is_blocked, created_at, visit_count, last_visit_at
		FROM clients
		WHERE master_id = $1 AND telegram_id = $2
	`, masterID, telegramID).Scan(
		&client.ID, &client.MasterID, &client.TelegramID, &client.TelegramUsername,
		&client.Name, &client.Phone, &client.ConsentGiven, &client.ConsentGivenAt,
		&client.NoBroadcast, &client.IsBlocked, &client.CreatedAt, &client.VisitCount, &client.LastVisitAt,
	)
	return client, err
}

func (r *ClientRepo) SaveConsent(ctx context.Context, clientID int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE clients SET consent_given=TRUE, consent_given_at=$2
		WHERE id=$1
	`, clientID, now)
	return err
}

func (r *ClientRepo) UpdateNamePhone(ctx context.Context, clientID int, name, phone string) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE clients SET name=$2, phone=$3 WHERE id=$1
	`, clientID, name, phone)
	return err
}

func (r *ClientRepo) MarkBlocked(ctx context.Context, masterID int, telegramID int64, blocked bool) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE clients SET is_blocked=$3
		WHERE master_id=$1 AND telegram_id=$2
	`, masterID, telegramID, blocked)
	return err
}

func (r *ClientRepo) SetNoBroadcast(ctx context.Context, clientID int, val bool) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE clients SET no_broadcast=$2 WHERE id=$1
	`, clientID, val)
	return err
}

func (r *ClientRepo) GetAllForBroadcast(ctx context.Context, masterID int, inactiveSince time.Time) ([]*models.Client, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
	SELECT c.id, c.master_id, c.telegram_id, c.telegram_username,
	       COALESCE(c.name,''), COALESCE(c.phone,''),
	       c.consent_given, c.consent_given_at,
	       c.no_broadcast, c.is_blocked, c.created_at
	FROM clients c
	LEFT JOIN (
		SELECT client_id, MAX(starts_at) AS last_visit
		FROM bookings
		WHERE status = 'completed'
		GROUP BY client_id
	) b ON b.client_id = c.id
	WHERE c.master_id = $1
	  AND c.consent_given = TRUE
	  AND c.no_broadcast = FALSE
	  AND c.is_blocked = FALSE
	  AND (
	    b.last_visit IS NULL
	    OR b.last_visit < $2
	  )
`, masterID, inactiveSince)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*models.Client
	for rows.Next() {
		cl := &models.Client{}
		if err := rows.Scan(
			&cl.ID, &cl.MasterID, &cl.TelegramID, &cl.TelegramUsername,
			&cl.Name, &cl.Phone, &cl.ConsentGiven, &cl.ConsentGivenAt,
			&cl.NoBroadcast, &cl.IsBlocked, &cl.CreatedAt,
		); err != nil {
			return nil, err
		}
		clients = append(clients, cl)
	}
	return clients, nil
}

func (r *ClientRepo) GetAllForMaster(ctx context.Context, masterID int) ([]*models.Client, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT id, master_id, telegram_id, telegram_username,
		       COALESCE(name,''), COALESCE(phone,''),
		       consent_given, consent_given_at,
		       no_broadcast, is_blocked, created_at, visit_count, last_visit_at
		FROM clients WHERE master_id=$1
		ORDER BY created_at DESC
	`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*models.Client
	for rows.Next() {
		cl := &models.Client{}
		if err := rows.Scan(
			&cl.ID, &cl.MasterID, &cl.TelegramID, &cl.TelegramUsername,
			&cl.Name, &cl.Phone, &cl.ConsentGiven, &cl.ConsentGivenAt,
			&cl.NoBroadcast, &cl.IsBlocked, &cl.CreatedAt, &cl.VisitCount, &cl.LastVisitAt,
		); err != nil {
			return nil, err
		}
		clients = append(clients, cl)
	}
	return clients, nil
}
func (r *ClientRepo) IncrementVisitCount(ctx context.Context, clientID int, lastVisitAt time.Time) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
        UPDATE clients 
        SET visit_count = visit_count + 1,
            last_visit_at = $2
        WHERE id = $1
    `, clientID, lastVisitAt)
	return err
}
