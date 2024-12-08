package main

import (
	"context"
	"database/sql"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) (*UserRepository, error) {
	return &UserRepository{
		db: db,
	}, nil
}

func (r *UserRepository) GetByAccessToken(ctx context.Context, token string) (*User, error) {
	cacheKey := "user_by_token:" + token
	if val, found := cache.Get(cacheKey); found {
		if u, ok := val.(*User); ok {
			return u, nil
		}
	}
	u := &User{}
	err := r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE access_token = ?", token).Scan(
		&u.ID, &u.Username, &u.Firstname, &u.Lastname, &u.DateOfBirth, &u.AccessToken, &u.InvitationCode, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	cache.Set(cacheKey, u, 1)
	return u, nil
}
