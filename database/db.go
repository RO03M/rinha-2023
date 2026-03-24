package database

import (
	"context"
	"fmt"
	"log"
	"rinha/pkg"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateDb() *pgxpool.Pool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username := pkg.GetRequiredEnv("DB_USER")
	password := pkg.GetRequiredEnv("DB_PASSWORD")
	host := pkg.GetRequiredEnv("DB_HOST")
	port := pkg.GetRequiredEnv("DB_PORT")
	dbName := pkg.GetRequiredEnv("DB_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, dbName)

	fmt.Println(connStr)

	config, err := pgxpool.ParseConfig(connStr)

	if err != nil {
		log.Fatal(err)
	}

	config.MaxConns = 10
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
