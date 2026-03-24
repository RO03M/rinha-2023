reset-db:
	docker compose down db --volumes && docker compose up -d db