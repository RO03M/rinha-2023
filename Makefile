reset-data:
	docker compose down redis db --volumes && docker compose up -d redis db