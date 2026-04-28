package repository

import (
	"beauty-bot/internal/models"
	"context"
	"time"

	"beauty-bot/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepo struct {
	db *pgxpool.Pool
}

func NewBookingRepo(db *pgxpool.Pool) *BookingRepo {
	return &BookingRepo{db: db}
}

func (r *BookingRepo) Create(ctx context.Context, b *models.Booking) (int, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	var id int
	err := r.db.QueryRow(ctx, `
		INSERT INTO bookings (master_id, client_id, service_id, starts_at, ends_at, status)
		VALUES ($1,$2,$3,$4,$5,'pending') RETURNING id
	`, b.MasterID, b.ClientID, b.ServiceID, b.StartsAt, b.EndsAt).Scan(&id)
	return id, err
}

func (r *BookingRepo) GetByID(ctx context.Context, id int) (*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	b := &models.Booking{}
	err := r.db.QueryRow(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.id=$1
	`, id).Scan(
		&b.ID, &b.MasterID, &b.ClientID, &b.ServiceID,
		&b.StartsAt, &b.EndsAt, &b.Status,
		&b.ConfirmedBy, &b.CancelReason,
		&b.Reminder24hSent, &b.Reminder2hSent, &b.ReviewRequested,
		&b.CreatedAt,
		&b.ClientName, &b.ClientPhone, &b.ClientTelegramID,
		&b.ServiceName, &b.ServicePrice, &b.ServiceDurationMin,
	)
	return b, err
}

func (r *BookingRepo) GetUpcomingForClient(ctx context.Context, masterID, clientID int) ([]*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.master_id=$1 AND b.client_id=$2
		  AND b.status NOT IN ('expired')
		  AND b.starts_at > NOW()
		ORDER BY b.starts_at
	`, masterID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *BookingRepo) GetForDay(ctx context.Context, masterID int, date time.Time) ([]*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.master_id=$1
		  AND b.starts_at >= $2 AND b.starts_at < $3
		ORDER BY b.starts_at
	`, masterID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *BookingRepo) IsSlotTaken(ctx context.Context, masterID int, startsAt, endsAt time.Time) (bool, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM bookings
		WHERE master_id=$1
		  AND status IN ('pending','confirmed')
		  AND starts_at < $3 AND ends_at > $2
	`, masterID, startsAt, endsAt).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// Also check blocked slots
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM blocked_slots
		WHERE master_id=$1
		  AND starts_at < $3 AND ends_at > $2
	`, masterID, startsAt, endsAt).Scan(&count)
	return count > 0, err
}

func (r *BookingRepo) Confirm(ctx context.Context, id int, by string) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE bookings SET status='confirmed', confirmed_by=$2 WHERE id=$1
	`, id, by)
	return err
}

func (r *BookingRepo) Cancel(ctx context.Context, id int, status, reason string) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE bookings SET status=$2, cancel_reason=$3 WHERE id=$1
	`, id, status, reason)
	return err
}

func (r *BookingRepo) MarkComplete(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE bookings SET status='completed' WHERE id=$1
	`, id)
	return err
}

func (r *BookingRepo) GetPendingForAutoConfirm(ctx context.Context, before time.Time) ([]*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.status='pending' AND b.created_at < $1
	`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *BookingRepo) GetNeedingReminder24h(ctx context.Context) ([]*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	from := time.Now().Add(23 * time.Hour)
	to := time.Now().Add(25 * time.Hour)
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.status='confirmed'
		  AND b.reminder_24h_sent=FALSE
		  AND b.starts_at BETWEEN $1 AND $2
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *BookingRepo) GetNeedingReminder2h(ctx context.Context) ([]*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	from := time.Now().Add(1*time.Hour + 45*time.Minute)
	to := time.Now().Add(2*time.Hour + 15*time.Minute)
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.status='confirmed'
		  AND b.reminder_2h_sent=FALSE
		  AND b.starts_at BETWEEN $1 AND $2
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *BookingRepo) GetNeedingReviewRequest(ctx context.Context) ([]*models.Booking, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	from := time.Now().Add(-4 * time.Hour)
	to := time.Now().Add(-2 * time.Hour)
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.master_id, b.client_id, b.service_id,
		       b.starts_at, b.ends_at, b.status,
		       COALESCE(b.confirmed_by,''), COALESCE(b.cancel_reason,''),
		       b.reminder_24h_sent, b.reminder_2h_sent, b.review_requested,
		       b.created_at,
		       COALESCE(c.name,''), COALESCE(c.phone,''), c.telegram_id,
		       s.name, s.price, s.duration_min
		FROM bookings b
		JOIN clients c ON c.id = b.client_id
		JOIN services s ON s.id = b.service_id
		WHERE b.status='confirmed'
		  AND b.review_requested=FALSE
		  AND b.ends_at BETWEEN $1 AND $2
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *BookingRepo) MarkReminder24hSent(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `UPDATE bookings SET reminder_24h_sent=TRUE WHERE id=$1`, id)
	return err
}

func (r *BookingRepo) MarkReminder2hSent(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `UPDATE bookings SET reminder_2h_sent=TRUE WHERE id=$1`, id)
	return err
}

func (r *BookingRepo) MarkReviewRequested(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `UPDATE bookings SET review_requested=TRUE WHERE id=$1`, id)
	return err
}

func (r *BookingRepo) GetStatsForMaster(ctx context.Context, masterID int, from, to time.Time) (total, completed, cancelled int, revenue int64, err error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status='completed'),
			COUNT(*) FILTER (WHERE status IN ('cancelled_by_client','cancelled_by_master')),
			COALESCE(SUM(s.price) FILTER (WHERE b.status='completed'), 0)
		FROM bookings b
		JOIN services s ON s.id = b.service_id
		WHERE b.master_id=$1 AND b.starts_at BETWEEN $2 AND $3
	`, masterID, from, to).Scan(&total, &completed, &cancelled, &revenue)
	return
}

func scanBookings(rows interface {
	Next() bool
	Scan(...any) error
}) ([]*models.Booking, error) {
	var bookings []*models.Booking
	for rows.Next() {
		b := &models.Booking{}
		if err := rows.Scan(
			&b.ID, &b.MasterID, &b.ClientID, &b.ServiceID,
			&b.StartsAt, &b.EndsAt, &b.Status,
			&b.ConfirmedBy, &b.CancelReason,
			&b.Reminder24hSent, &b.Reminder2hSent, &b.ReviewRequested,
			&b.CreatedAt,
			&b.ClientName, &b.ClientPhone, &b.ClientTelegramID,
			&b.ServiceName, &b.ServicePrice, &b.ServiceDurationMin,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func (r *BookingRepo) GetActiveForClient(ctx context.Context, masterID int, clientID int) ([]*models.Booking, error) {
	rows, err := r.db.Query(ctx, `
        SELECT id, service_id, service_name, service_price, service_duration_min,
               starts_at, status
        FROM bookings
        WHERE master_id = $1
          AND client_id = $2
          AND status IN ('pending', 'confirmed')
        ORDER BY starts_at
    `, masterID, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		b := &models.Booking{}
		err := rows.Scan(
			&b.ID,
			&b.ServiceID,
			&b.ServiceName,
			&b.ServicePrice,
			&b.ServiceDurationMin,
			&b.StartsAt,
			&b.Status,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bookings, nil
}
