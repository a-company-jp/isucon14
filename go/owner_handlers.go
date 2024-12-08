package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/oklog/ulid/v2"
)

const (
	initialFare     = 500
	farePerDistance = 100
)

type ownerPostOwnersRequest struct {
	Name string `json:"name"`
}

type ownerPostOwnersResponse struct {
	ID                 string `json:"id"`
	ChairRegisterToken string `json:"chair_register_token"`
}

func ownerPostOwners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &ownerPostOwnersRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("some of required fields(name) are empty"))
		return
	}

	ownerID := ulid.Make().String()
	accessToken := secureRandomStr(32)
	chairRegisterToken := secureRandomStr(32)

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO owners (id, name, access_token, chair_register_token) VALUES (?, ?, ?, ?)",
		ownerID, req.Name, accessToken, chairRegisterToken,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to insert owner: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Path:  "/",
		Name:  "owner_session",
		Value: accessToken,
	})

	writeJSON(w, http.StatusCreated, &ownerPostOwnersResponse{
		ID:                 ownerID,
		ChairRegisterToken: chairRegisterToken,
	})
}

type chairSales struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Sales int    `json:"sales"`
}

type modelSales struct {
	Model string `json:"model"`
	Sales int    `json:"sales"`
}

type ownerGetSalesResponse struct {
	TotalSales int          `json:"total_sales"`
	Chairs     []chairSales `json:"chairs"`
	Models     []modelSales `json:"models"`
}

func ownerGetSales(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	since := time.Unix(0, 0)
	until := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if r.URL.Query().Get("since") != "" {
		parsed, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		since = time.UnixMilli(parsed)
	}
	if r.URL.Query().Get("until") != "" {
		parsed, err := strconv.ParseInt(r.URL.Query().Get("until"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		until = time.UnixMilli(parsed)
	}

	owner := ctx.Value("owner").(*Owner)

	chairs := []Chair{}
	if err := db.SelectContext(ctx, &chairs, "SELECT * FROM chairs WHERE owner_id = ?", owner.ID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to get chairs: %w", err))
		return
	}

	chairIDs := []string{}
	for _, chair := range chairs {
		chairIDs = append(chairIDs, chair.ID)
	}

	type rideSales struct {
		ChairID string `db:"chair_id"`
		Sales   int    `db:"sales"`
	}

	rideSalesData := []rideSales{}
	query := `
		SELECT rides.chair_id,
		       SUM(?) + SUM(ABS(rides.pickup_latitude - rides.destination_latitude) + ABS(rides.pickup_longitude - rides.destination_longitude) * ?) AS sales
		FROM rides
		JOIN ride_statuses ON rides.id = ride_statuses.ride_id
		WHERE rides.chair_id IN (?) AND ride_statuses.status = 'COMPLETED' AND ride_statuses.updated_at BETWEEN ? AND ?
		GROUP BY rides.chair_id
	`
	query, args, err := sqlx.In(query, initialFare, farePerDistance, chairIDs, since, until)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to create query: %w", err))
		return
	}
	query = db.Rebind(query)

	if err := db.SelectContext(ctx, &rideSalesData, query, args...); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to get ride sales: %w", err))
		return
	}

	// 売上データをマッピング
	chairSalesMap := make(map[string]int)
	for _, data := range rideSalesData {
		chairSalesMap[data.ChairID] = data.Sales
	}

	// レスポンス構造体を作成
	res := ownerGetSalesResponse{
		TotalSales: 0,
	}

	modelSalesByModel := map[string]int{}
	for _, chair := range chairs {
		sales := chairSalesMap[chair.ID]
		res.TotalSales += sales

		res.Chairs = append(res.Chairs, chairSales{
			ID:    chair.ID,
			Name:  chair.Name,
			Sales: sales,
		})

		modelSalesByModel[chair.Model] += sales
	}

	// モデル別売上を作成
	for model, sales := range modelSalesByModel {
		res.Models = append(res.Models, modelSales{
			Model: model,
			Sales: sales,
		})
	}

	// レスポンスを返却
	writeJSON(w, http.StatusOK, res)
}

func sumSales(rides []Ride) int {
	sale := 0
	for _, ride := range rides {
		sale += calculateSale(ride)
	}
	return sale
}

func calculateSale(ride Ride) int {
	return calculateFare(ride.PickupLatitude, ride.PickupLongitude, ride.DestinationLatitude, ride.DestinationLongitude)
}

type chairWithDetail struct {
	ID                     string       `db:"id"`
	OwnerID                string       `db:"owner_id"`
	Name                   string       `db:"name"`
	AccessToken            string       `db:"access_token"`
	Model                  string       `db:"model"`
	IsActive               bool         `db:"is_active"`
	CreatedAt              time.Time    `db:"created_at"`
	UpdatedAt              time.Time    `db:"updated_at"`
	TotalDistance          int          `db:"total_distance"`
	TotalDistanceUpdatedAt sql.NullTime `db:"total_distance_updated_at"`
}

type ownerGetChairResponse struct {
	Chairs []ownerGetChairResponseChair `json:"chairs"`
}

type ownerGetChairResponseChair struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Model                  string `json:"model"`
	Active                 bool   `json:"active"`
	RegisteredAt           int64  `json:"registered_at"`
	TotalDistance          int    `json:"total_distance"`
	TotalDistanceUpdatedAt *int64 `json:"total_distance_updated_at,omitempty"`
}

func ownerGetChairs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	owner := ctx.Value("owner").(*Owner)

	chairs := []chairWithDetail{}
	if err := db.SelectContext(ctx, &chairs, `
        SELECT
            id,
            owner_id,
            name,
            access_token,
            model,
            is_active,
            created_at,
            updated_at,
            IFNULL(chairs.total_distance, 0) AS total_distance,
            total_distance_updated_at
        FROM chairs
        WHERE owner_id = ?
    `, owner.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	res := ownerGetChairResponse{}
	for _, chair := range chairs {
		c := ownerGetChairResponseChair{
			ID:            chair.ID,
			Name:          chair.Name,
			Model:         chair.Model,
			Active:        chair.IsActive,
			RegisteredAt:  chair.CreatedAt.UnixMilli(),
			TotalDistance: chair.TotalDistance,
		}
		if chair.TotalDistanceUpdatedAt.Valid {
			t := chair.TotalDistanceUpdatedAt.Time.UnixMilli()
			c.TotalDistanceUpdatedAt = &t
		}
		res.Chairs = append(res.Chairs, c)
	}
	writeJSON(w, http.StatusOK, res)
}
