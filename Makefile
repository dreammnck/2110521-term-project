.PHONY: up down logs proto test smoke clean

up:
	docker compose up --build -d

down:
	docker compose down

clean:
	docker compose down -v

logs:
	docker compose logs -f --tail=100

# Regenerates backend/gen from proto/ (tools are installed into .tools/).
proto:
	cd proto && PATH=$(CURDIR)/.tools:$$PATH buf generate

test:
	cd backend && go vet ./... && go test ./...

smoke:
	./scripts/smoke.sh
