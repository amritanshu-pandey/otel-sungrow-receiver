build:
	docker build -t otel-sungrow-receiver:1 .
run:
	docker compose up -d
stop:
	docker compose down
