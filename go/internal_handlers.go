package main

import (
	"database/sql"
	"errors"
	"net/http"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ride := &Ride{}
	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1 FOR UPDATE`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// 利用可能な椅子を効率的に取得（例：最も近い椅子）
	query := `
        SELECT c.*
        FROM chairs c
        JOIN chair_locations cl ON c.id = cl.chair_id
        WHERE c.is_active = TRUE
          AND NOT EXISTS (
              SELECT 1
              FROM rides r
              JOIN ride_statuses rs ON r.id = rs.ride_id
              WHERE r.chair_id = c.id
                AND rs.status != 'COMPLETED'
          )
        ORDER BY ABS(cl.latitude - ?) + ABS(cl.longitude - ?) ASC
        LIMIT 1
    `

	matched := &Chair{}
	if err := db.GetContext(ctx, matched, query, ride.PickupLatitude, ride.PickupLongitude); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
