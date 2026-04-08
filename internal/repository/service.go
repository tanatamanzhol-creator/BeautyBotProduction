package repository

import (
	"context"
	"beauty-bot/internal/models"

	"beauty-bot/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceRepo struct {
	db *pgxpool.Pool
}

func NewServiceRepo(db *pgxpool.Pool) *ServiceRepo {
	return &ServiceRepo{db: db}
}

func (r *ServiceRepo) GetActive(ctx context.Context, masterID int) ([]*models.Service, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT id, master_id, category_id, name, price, price_from, duration_min, is_active, sort_order
		FROM services
		WHERE master_id=$1 AND is_active=TRUE
		ORDER BY sort_order, id
	`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		s := &models.Service{}
		if err := rows.Scan(&s.ID, &s.MasterID, &s.CategoryID, &s.Name,
			&s.Price, &s.PriceFrom, &s.DurationMin, &s.IsActive, &s.SortOrder); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *ServiceRepo) GetAll(ctx context.Context, masterID int) ([]*models.Service, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT id, master_id, category_id, name, price, price_from, duration_min, is_active, sort_order
		FROM services WHERE master_id=$1
		ORDER BY sort_order, id
	`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		s := &models.Service{}
		if err := rows.Scan(&s.ID, &s.MasterID, &s.CategoryID, &s.Name,
			&s.Price, &s.PriceFrom, &s.DurationMin, &s.IsActive, &s.SortOrder); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *ServiceRepo) GetByID(ctx context.Context, id int) (*models.Service, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	s := &models.Service{}
	err := r.db.QueryRow(ctx, `
		SELECT id, master_id, category_id, name, price, price_from, duration_min, is_active, sort_order
		FROM services WHERE id=$1
	`, id).Scan(&s.ID, &s.MasterID, &s.CategoryID, &s.Name,
		&s.Price, &s.PriceFrom, &s.DurationMin, &s.IsActive, &s.SortOrder)
	return s, err
}

func (r *ServiceRepo) Create(ctx context.Context, s *models.Service) (int, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	var id int
	err := r.db.QueryRow(ctx, `
		INSERT INTO services (master_id, category_id, name, price, price_from, duration_min, sort_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, s.MasterID, s.CategoryID, s.Name, s.Price, s.PriceFrom, s.DurationMin, s.SortOrder).Scan(&id)
	return id, err
}

func (r *ServiceRepo) Update(ctx context.Context, s *models.Service) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE services SET name=$2, price=$3, price_from=$4, duration_min=$5, category_id=$6
		WHERE id=$1
	`, s.ID, s.Name, s.Price, s.PriceFrom, s.DurationMin, s.CategoryID)
	return err
}

func (r *ServiceRepo) SetActive(ctx context.Context, id int, active bool) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `UPDATE services SET is_active=$2 WHERE id=$1`, id, active)
	return err
}

func (r *ServiceRepo) Delete(ctx context.Context, id int) error {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	_, err := r.db.Exec(ctx, `DELETE FROM services WHERE id=$1`, id)
	return err
}

func (r *ServiceRepo) GetCategories(ctx context.Context, masterID int) ([]*models.ServiceCategory, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		SELECT id, master_id, name, sort_order
		FROM service_categories WHERE master_id=$1
		ORDER BY sort_order, id
	`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*models.ServiceCategory
	for rows.Next() {
		c := &models.ServiceCategory{}
		if err := rows.Scan(&c.ID, &c.MasterID, &c.Name, &c.SortOrder); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, nil
}

func (r *ServiceRepo) HasActiveBookings(ctx context.Context, serviceID int) (bool, error) {
	ctx, cancel := db.NewContext(ctx)
	defer cancel()

	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM bookings
		WHERE service_id=$1 AND status IN ('pending','confirmed')
	`, serviceID).Scan(&count)
	return count > 0, err
}
