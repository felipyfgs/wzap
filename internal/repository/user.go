package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"wzap/internal/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO "wzUsers" ("id", "name", "token", "webhook", "events", "expiration", "proxyUrl", "history", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Name, user.Token, user.Webhook, user.Events,
		user.Expiration, user.ProxyUrl, user.History, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]model.User, error) {
	query := `SELECT "id", "name", "token", "webhook", "events", "expiration",
		COALESCE("proxyUrl", ''), "history", "createdAt", "updatedAt"
		FROM "wzUsers" ORDER BY "createdAt" DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Token, &u.Webhook, &u.Events,
			&u.Expiration, &u.ProxyUrl, &u.History, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT "id", "name", "token", "webhook", "events", "expiration",
		COALESCE("proxyUrl", ''), "history", "createdAt", "updatedAt"
		FROM "wzUsers" WHERE "id" = $1`

	var u model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Name, &u.Token, &u.Webhook, &u.Events,
		&u.Expiration, &u.ProxyUrl, &u.History, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) FindByToken(ctx context.Context, token string) (*model.User, error) {
	query := `SELECT "id", "name", "token", "webhook", "events", "expiration",
		COALESCE("proxyUrl", ''), "history", "createdAt", "updatedAt"
		FROM "wzUsers" WHERE "token" = $1`

	var u model.User
	err := r.db.QueryRow(ctx, query, token).Scan(
		&u.ID, &u.Name, &u.Token, &u.Webhook, &u.Events,
		&u.Expiration, &u.ProxyUrl, &u.History, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found with this token: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "wzUsers" WHERE "id" = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user %s: %w", id, err)
	}
	return nil
}
