url_go:
	docker compose up --force-recreate --build -d

migrate:
	migrate create -ext sql -dir ./migrations -seq initial
