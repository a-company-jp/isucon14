package main

import (
	"context"
	"database/sql"
)

type OwnerRepository struct {
	db *sql.DB
}

func NewOwnerRepository(db *sql.DB) (*OwnerRepository, error) {
	return &OwnerRepository{
		db: db,
	}, nil
}

func (r *OwnerRepository) GetByAccessToken(ctx context.Context, token string) (*Owner, error) {
	cacheKey := "owner_by_token:" + token
	if val, found := cache.Get(cacheKey); found {
		if o, ok := val.(*Owner); ok {
			return o, nil
		}
	}
	o := &Owner{}
	err := r.db.QueryRowContext(ctx, "SELECT * FROM owners WHERE access_token = ?", token).Scan(
		&o.ID, &o.Name, &o.AccessToken, &o.ChairRegisterToken, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	cache.Set(cacheKey, o, 1)
	return o, nil
}
