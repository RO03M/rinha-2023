package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateDb() *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://admin:admin@localhost:5432/rinha?sslmode=disable"

	config, err := pgxpool.ParseConfig(connStr)

	if err != nil {
		log.Fatal(err)
	}

	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	db, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Initialized postgres connection: ", connStr)

	return db
}
