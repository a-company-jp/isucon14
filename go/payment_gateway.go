package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var erroredUpstream = errors.New("errored upstream")

type paymentGatewayPostPaymentRequest struct {
	Amount int `json:"amount"`
}

type paymentGatewayGetPaymentsResponseOne struct {
	Amount int    `json:"amount"`
	Status string `json:"status"`
}

func requestPaymentGatewayPostPayment(ctx context.Context, paymentGatewayURL string, token string, param *paymentGatewayPostPaymentRequest, retrieveRidesOrderByCreatedAtAsc func() ([]Ride, error)) error {
	b, err := json.Marshal(param)
	if err != nil {
		return err
	}

	// リトライ回数
	const maxRetries = 5
	retry := 0
	for {
		// リクエストの実行
		err := func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, paymentGatewayURL+"/payments", bytes.NewBuffer(b))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			// POSTリクエスト成功ステータスコードの確認
			if res.StatusCode == http.StatusNoContent || res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
				// 成功した場合はループを終了
				return nil
			}

			// エラーが返ってきても成功している場合があるので、社内決済マイクロサービスに問い合わせ
			getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, paymentGatewayURL+"/payments", bytes.NewBuffer([]byte{}))
			if err != nil {
				return err
			}
			getReq.Header.Set("Authorization", "Bearer "+token)

			getRes, err := http.DefaultClient.Do(getReq)
			if err != nil {
				return err
			}
			defer getRes.Body.Close()

			// GET /payments は障害と関係なく200が返るので、200以外は回復不能なエラーとする
			if getRes.StatusCode != http.StatusOK {
				// レスポンス内容をエラーメッセージに追加
				body, _ := io.ReadAll(getRes.Body)
				return fmt.Errorf("[GET /payments] unexpected status code (%d). Response: %s", getRes.StatusCode, string(body))
			}

			var payments []paymentGatewayGetPaymentsResponseOne
			if err := json.NewDecoder(getRes.Body).Decode(&payments); err != nil {
				return err
			}

			// Ride の取得
			rides, err := retrieveRidesOrderByCreatedAtAsc()
			if err != nil {
				return err
			}

			// Ride と Payment の数が一致しない場合エラー
			if len(rides) != len(payments) {
				return fmt.Errorf("unexpected number of payments: %d != %d. %w", len(rides), len(payments), erroredUpstream)
			}

			return nil
		}()
		if err != nil {
			if retry < maxRetries {
				retry++
				// 指数バックオフ
				time.Sleep(time.Duration(1<<retry) * time.Second) // 1, 2, 4, 8, 16秒
				continue
			} else {
				return err
			}
		}
		break
	}

	return nil
}
