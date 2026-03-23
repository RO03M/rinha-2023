package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CreatePerson struct {
	Id       string   `json:"id"`
	Name     string   `json:"nome"`
	Nickname string   `json:"apelido"`
	Birthday string   `json:"nascimento"`
	Stack    []string `json:"stack"`
}

type Person struct {
	Id       string    `json:"id"`
	Name     string    `json:"nome"`
	Nickname string    `json:"apelido"`
	Birthday time.Time `json:"nascimento"`
	Stack    []string  `json:"stack"`
}

type PersonService struct {
	db *pgxpool.Pool
}

func (service *PersonService) CreatePerson(ctx context.Context, name string, nickname string, birthday string, stack []string) (string, error) {
	var id string

	query := `
		INSERT INTO people (nickname, name, birthday, stack, search)
		VALUES (
			$1::text,
			$2::text,
			$3,
			$4,
			to_tsvector('simple',
				$2::text || ' ' ||
				$1::text || ' ' ||
				array_to_string($4::text[], ' ')
			)
		)
		RETURNING id
	`

	err := service.db.QueryRow(
		ctx,
		query,
		nickname,
		name,
		birthday,
		stack,
	).Scan(&id)

	return id, err
}

func (service *PersonService) GetPersonById(ctx context.Context, id string) (Person, error) {
	var person Person

	err := service.db.QueryRow(
		ctx,
		"SELECT id, name, nickname, birthday, stack FROM people WHERE id = $1",
		id,
	).Scan(
		&person.Id,
		&person.Name,
		&person.Nickname,
		&person.Birthday,
		&person.Stack,
	)

	if err != nil {
		return Person{}, err
	}

	return person, nil
}

func (service *PersonService) GetPeople(ctx context.Context, limit int) ([]Person, error) {
	var people []Person = make([]Person, 0)

	rows, err := service.db.Query(
		ctx,
		`SELECT id, name, nickname, birthday, stack
		FROM people
		LIMIT %1`,
		limit,
	)

	if err != nil {
		return []Person{}, err
	}

	for rows.Next() {
		var person Person

		err := rows.Scan(
			&person.Id,
			&person.Name,
			&person.Nickname,
			&person.Birthday,
			&person.Stack,
		)

		if err != nil {
			return []Person{}, err
		}

		people = append(people, person)
	}

	return people, nil
}

func (service *PersonService) GetPeopleByTerm(ctx context.Context, term string) ([]Person, error) {
	var people []Person = make([]Person, 0)

	query := `
	SELECT id, name, nickname, birthday, stack
	FROM people
	WHERE search @@ plainto_tsquery('simple', $1)
	LIMIT 50
	`

	rows, err := service.db.Query(
		ctx,
		query,
		term,
	)

	if err != nil {
		return []Person{}, err
	}

	for rows.Next() {
		var person Person

		err := rows.Scan(
			&person.Id,
			&person.Name,
			&person.Nickname,
			&person.Birthday,
			&person.Stack,
		)

		if err != nil {
			return []Person{}, err
		}

		people = append(people, person)
	}

	return people, nil
}

func (service *PersonService) Count(ctx context.Context) int {
	var total int

	err := service.db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&total)

	if err != nil {
		fmt.Println(err)
		return -1
	}

	return total
}
