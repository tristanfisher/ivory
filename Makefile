test:
	docker-compose up --detach
	go test ./...
	docker-compose down