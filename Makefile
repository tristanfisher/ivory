test:
	# prefix with a docker-compose down
	docker-compose down ivory_tester --volumes
	docker-compose up ivory_tester --detach
	go test ./...
	docker-compose down ivory_tester --volumes
