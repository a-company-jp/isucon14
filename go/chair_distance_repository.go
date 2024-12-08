package main

import (
	"context"
	"database/sql"
	"github.com/dgraph-io/ristretto"
	"time"
)

type ChairDistance struct {
	ChairID              string
	TotalDistance        int
	TotalDistanceUpdated time.Time
}

type ChairDistanceRepository struct {
	db    *sql.DB
	cache *ristretto.Cache
}

// NewChairDistanceRepository はキャッシュ付きのレポジトリを生成
func NewChairDistanceRepository(db *sql.DB) (*ChairDistanceRepository, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e4,
		MaxCost:     1 << 25,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}
	return &ChairDistanceRepository{
		db:    db,
		cache: cache,
	}, nil
}

// GetTotalDistance はキャッシュから合計距離を取得し、キャッシュに無ければDBを参照して計算する
func (r *ChairDistanceRepository) GetTotalDistance(ctx context.Context, chairID string) (*ChairDistance, error) {
	if val, found := r.cache.Get("chair_distance:" + chairID); found {
		if dist, ok := val.(*ChairDistance); ok {
			return dist, nil
		}
	}

	// キャッシュに無い場合はDBから合計距離を計算
	// ここでは簡略化のため、chair_locationsテーブルの全履歴を走査して距離計算する例を示します。
	// 実際には初回のみこの処理を行い、以降はCreate/Updateイベント時に差分更新で済ませることを想定。
	rows, err := r.db.QueryContext(ctx, `
		SELECT latitude, longitude, created_at FROM chair_locations WHERE chair_id = ? ORDER BY created_at ASC
	`, chairID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prevLat, prevLon int
	var totalDist int
	first := true
	var lastTime time.Time

	for rows.Next() {
		var lat, lon int
		var createdAt time.Time
		if err := rows.Scan(&lat, &lon, &createdAt); err != nil {
			return nil, err
		}
		if first {
			prevLat, prevLon = lat, lon
			first = false
		} else {
			totalDist += abs(lat-prevLat) + abs(lon-prevLon)
			prevLat, prevLon = lat, lon
		}
		lastTime = createdAt
	}

	dist := &ChairDistance{
		ChairID:              chairID,
		TotalDistance:        totalDist,
		TotalDistanceUpdated: lastTime,
	}
	r.cache.Set("chair_distance:"+chairID, dist, 1)

	return dist, nil
}

// UpdateDistanceはchair_locationsに新たなレコードが追加された際に差分だけ更新
// 実際にはChairLocationRepositoryから呼び出し、前回の最新地点との差分計算でtotalDistを増加させる運用が望ましい
func (r *ChairDistanceRepository) UpdateDistance(ctx context.Context, loc *ChairLocation) error {
	distVal, err := r.GetTotalDistance(ctx, loc.ChairID)
	if err != nil {
		return err
	}

	// 最終更新日時以降に追加されたlocとの差分計算(簡略化)
	// ここは前回地点(prevLat, prevLon)をキャッシュするなど工夫するとよい
	// 例: 前回計算済みの最後の座標を別途保存しておき、その差分のみ足す
	//
	// 簡易的な例：
	prevLoc, err := r.getLastLocationBefore(ctx, loc.ChairID, distVal.TotalDistanceUpdated)
	if err != nil {
		return err
	}
	if prevLoc != nil {
		diff := abs(loc.Latitude-prevLoc.Latitude) + abs(loc.Longitude-prevLoc.Longitude)
		distVal.TotalDistance += diff
		distVal.TotalDistanceUpdated = loc.CreatedAt
		r.cache.Set("chair_distance:"+loc.ChairID, distVal, 1)
	}
	return nil
}

// 前回更新以降で一番新しいロケーションを取得
// 実際にはこうしたヘルパーを設けることで差分更新を実現する
func (r *ChairDistanceRepository) getLastLocationBefore(ctx context.Context, chairID string, since time.Time) (*ChairLocation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, chair_id, latitude, longitude, created_at 
		FROM chair_locations 
		WHERE chair_id = ? AND created_at <= ?
		ORDER BY created_at DESC LIMIT 1
	`, chairID, since)
	loc := &ChairLocation{}
	err := row.Scan(&loc.ID, &loc.ChairID, &loc.Latitude, &loc.Longitude, &loc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return loc, nil
}
