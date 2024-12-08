package main

import (
	"database/sql"
)

type ChairLocationRepository struct {
	db *sql.DB
}

func NewChairLocationRepository(db *sql.DB) (*ChairRepository, error) {
	return &ChairRepository{
		db: db,
	}, nil
}

//
//// GetChairsByOwnerID は特定のオーナーIDに紐づくChair一覧を取得します
//// キャッシュヒット時は即返却し、ミス時はDBから取得してキャッシュします。
//func (r *ChairRepository) GetChairsByOwnerID(ctx context.Context, ownerID string) ([]Chair, error) {
//	cacheKey := "chair_list_by_owner:" + ownerID
//
//	if val, found := cache.Get(cacheKey); found {
//		if chairs, ok := val.([]Chair); ok {
//			return chairs, nil
//		}
//	}
//
//	chairs := []Chair{}
//	err := r.selectChairsByOwnerID(ctx, ownerID, &chairs)
//	if err != nil {
//		return nil, err
//	}
//
//	cache.Set(cacheKey, chairs, 1)
//	return chairs, nil
//}
//
//// 更新等でキャッシュを無効化する際に使用できる
//func (r *ChairRepository) InvalidateCacheByOwnerID(ownerID string) {
//	cacheKey := "chair_list_by_owner:" + ownerID
//	cache.Del(cacheKey)
//}
//
//func (r *ChairRepository) selectChairsByOwnerID(ctx context.Context, ownerID string, dest *[]Chair) error {
//	rows, err := r.db.QueryContext(ctx, `
//		SELECT id, owner_id, name, model, is_active, access_token, created_at, updated_at
//		FROM chairs
//		WHERE owner_id = ?
//	`, ownerID)
//	if err != nil {
//		return err
//	}
//	defer rows.Close()
//
//	var results []Chair
//	for rows.Next() {
//		var c Chair
//		if err := rows.Scan(&c.ID, &c.OwnerID, &c.Name, &c.Model, &c.IsActive, &c.AccessToken, &c.CreatedAt, &c.UpdatedAt); err != nil {
//			return err
//		}
//		results = append(results, c)
//	}
//	if err := rows.Err(); err != nil {
//		return err
//	}
//
//	*dest = results
//	return nil
//}
