.PHONY: swagger
swagger:
	swag init -g internal/app/handlers.go

.PHONY: swagg-fmt
swagg-fmt:
	swag fmt --dir internal/app

.PHONY: test
test: 
	go test ./test -v

.PHONY: up
up:
	docker compose up --build