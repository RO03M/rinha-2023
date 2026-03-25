package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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

type insertRequest struct {
	person CreatePerson
}

type PersonService struct {
	db        *pgxpool.Pool
	queue     chan insertRequest
	insertMap map[string]insertRequest
	redis     *redis.Client
}

func NewPersonService(db *pgxpool.Pool, redis *redis.Client) *PersonService {
	service := &PersonService{
		db:        db,
		redis:     redis,
		queue:     make(chan insertRequest, 10000),
		insertMap: map[string]insertRequest{},
	}

	go service.tickWorker()

	return service
}
func (service *PersonService) ClaimNickname(nickname string) (bool, error) {
	ok, err := service.redis.SetArgs(context.Background(), "nick:"+nickname, 1, redis.SetArgs{
		Mode: "NX",
		TTL:  0,
	}).Result()

	return ok == "OK", err
}

func (service *PersonService) tickWorker() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case req := <-service.queue:
			if _, exists := service.insertMap[req.person.Nickname]; !exists {
				service.insertMap[req.person.Nickname] = req
			}
		case <-ticker.C:
			service.bulkInsert()
			// break
			// fmt.Println("tick it motherfucker")
		}
	}
}

func (service *PersonService) bulkInsert() {
	if len(service.insertMap) == 0 {
		return
	}

	args := make([]any, 0, len(service.insertMap)*5)
	placeholders := make([]string, 0, len(service.insertMap))

	i := 0

	for _, person := range service.insertMap {
		base := i * 5
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d::uuid, $%d::text, $%d::text, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5,
		))
		args = append(args, person.person.Id, person.person.Name, person.person.Nickname, person.person.Birthday, person.person.Stack)

		i++

		delete(service.insertMap, person.person.Nickname)
	}

	query := "INSERT INTO people (id, name, nickname, birthday, stack) VALUES " +
		strings.Join(placeholders, ", ")

	_, err := service.db.Exec(context.Background(), query, args...)
	if err != nil {
		fmt.Println("batch insert error:", err)
	}
}

func (service *PersonService) CachePerson(ctx context.Context, person CreatePerson) error {
	data, err := sonic.Marshal(person)
	if err != nil {
		return err
	}
	return service.redis.Set(ctx, "person:"+person.Id, data, 5*time.Minute).Err()
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
