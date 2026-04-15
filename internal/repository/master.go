package repository

import (
	"context"

	"beauty-bot/internal/db"
	"beauty-bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MasterRepo struct {
	db *pgxpool.Pool
}

func NewMasterRepo(pool *pgxpool.Pool) *MasterRepo {
	return &MasterRepo{db: pool}
}

func (r *MasterRepo) GetAll(ctx context.Context) ([]*models.Master, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT id, name, address, client_bot_token, admin_bot_token,
		       COALESCE(client_bot_username,''), COALESCE(admin_bot_username,''),
		       COALESCE(welcome_text,''), is_active,
		       COALESCE(master_telegram_id, 0),
		       trial_started_at, trial_ends_at, paid_until,
		       slot_interval_min, min_hours_before_booking, cancel_limit_hours,
		       mon_start, mon_end, tue_start, tue_end,
		       wed_start, wed_end, thu_start, thu_end,
		       fri_start, fri_end, sat_start, sat_end,
		       sun_start, sun_end, created_at
		FROM masters WHERE is_active = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var masters []*models.Master
	for rows.Next() {
		m := &models.Master{}
		err := rows.Scan(
			&m.ID, &m.Name, &m.Address, &m.ClientBotToken, &m.AdminBotToken,
			&m.ClientBotUsername, &m.AdminBotUsername, &m.WelcomeText,
			&m.IsActive, &m.MasterTelegramID,
			&m.TrialStartedAt, &m.TrialEndsAt, &m.PaidUntil,
			&m.SlotIntervalMin, &m.MinHoursBeforeBooking, &m.CancelLimitHours,
			&m.Schedule.Mon.Start, &m.Schedule.Mon.End,
			&m.Schedule.Tue.Start, &m.Schedule.Tue.End,
			&m.Schedule.Wed.Start, &m.Schedule.Wed.End,
			&m.Schedule.Thu.Start, &m.Schedule.Thu.End,
			&m.Schedule.Fri.Start, &m.Schedule.Fri.End,
			&m.Schedule.Sat.Start, &m.Schedule.Sat.End,
			&m.Schedule.Sun.Start, &m.Schedule.Sun.End,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		masters = append(masters, m)
	}
	return masters, nil
}

func (r *MasterRepo) GetByID(ctx context.Context, id int) (*models.Master, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	m := &models.Master{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, address, client_bot_token, admin_bot_token,
		       COALESCE(client_bot_username,''), COALESCE(admin_bot_username,''),
		       COALESCE(welcome_text,''), is_active,
		       COALESCE(master_telegram_id, 0),
		       trial_started_at, trial_ends_at, paid_until,
		       slot_interval_min, min_hours_before_booking, cancel_limit_hours,
		       mon_start, mon_end, tue_start, tue_end,
		       wed_start, wed_end, thu_start, thu_end,
		       fri_start, fri_end, sat_start, sat_end,
		       sun_start, sun_end, COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(poi_id, ''), created_at
		FROM masters WHERE id = $1
	`, id).Scan(
		&m.ID, &m.Name, &m.Address, &m.ClientBotToken, &m.AdminBotToken,
		&m.ClientBotUsername, &m.AdminBotUsername, &m.WelcomeText,
		&m.IsActive, &m.MasterTelegramID,
		&m.TrialStartedAt, &m.TrialEndsAt, &m.PaidUntil,
		&m.SlotIntervalMin, &m.MinHoursBeforeBooking, &m.CancelLimitHours,
		&m.Schedule.Mon.Start, &m.Schedule.Mon.End,
		&m.Schedule.Tue.Start, &m.Schedule.Tue.End,
		&m.Schedule.Wed.Start, &m.Schedule.Wed.End,
		&m.Schedule.Thu.Start, &m.Schedule.Thu.End,
		&m.Schedule.Fri.Start, &m.Schedule.Fri.End,
		&m.Schedule.Sat.Start, &m.Schedule.Sat.End,
		&m.Schedule.Sun.Start, &m.Schedule.Sun.End,
		&m.Latitude,
		&m.Longitude,
		&m.PoiID,
		&m.CreatedAt,
	)
	return m, err
}

func (r *MasterRepo) Activate(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()
	_, err := r.db.Exec(ctx, `UPDATE masters SET is_active = TRUE WHERE id = $1`, id)
	return err
}

func (r *MasterRepo) Deactivate(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()
	_, err := r.db.Exec(ctx, `UPDATE masters SET is_active = FALSE WHERE id = $1`, id)
	return err
}

func (r *MasterRepo) UpdateSchedule(ctx context.Context, masterID int, s models.WeekSchedule) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		UPDATE masters SET
			mon_start=$2, mon_end=$3, tue_start=$4, tue_end=$5,
			wed_start=$6, wed_end=$7, thu_start=$8, thu_end=$9,
			fri_start=$10, fri_end=$11, sat_start=$12, sat_end=$13,
			sun_start=$14, sun_end=$15
		WHERE id = $1
	`, masterID,
		s.Mon.Start, s.Mon.End, s.Tue.Start, s.Tue.End,
		s.Wed.Start, s.Wed.End, s.Thu.Start, s.Thu.End,
		s.Fri.Start, s.Fri.End, s.Sat.Start, s.Sat.End,
		s.Sun.Start, s.Sun.End,
	)
	return err
}

func (r *MasterRepo) UpdateSettings(ctx context.Context, masterID, slotInterval, minHours, cancelLimit int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		UPDATE masters SET
			slot_interval_min=$2,
			min_hours_before_booking=$3,
			cancel_limit_hours=$4
		WHERE id = $1
	`, masterID, slotInterval, minHours, cancelLimit)
	return err
}
