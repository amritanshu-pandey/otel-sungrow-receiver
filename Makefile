build:
	docker build -t otel-sungrow-receiver:1 .
run:
	docker compose up -d
stop:
	docker compose down
logs:
	docker compose logs -f
