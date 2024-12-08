DB_NAME = "isupipe"
DB_USER = "isucon"
DB_PASS = "isucon"
DB_NAME = "isuride"
# mysql
DB_URI = "mysql://$(DB_USER):$(DB_PASS)@tcp(localhost:3306)/$(DB_NAME)?parseTime=true"

new: ## マイグレーションファイル作成
	migrate create -ext sql -dir ./migrate -seq "migrate"

up: ## マイグレーション実行
	migrate -path ./migrate -database $(DB_URI) up
