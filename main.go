package main

import (
	"errors"
	"fmt"
	"log"
	"rinha/database"
	"rinha/pkg"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func isValidDate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9} {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	month := (int(s[5]-'0') * 10) + int(s[6]-'0')
	day := (int(s[8]-'0') * 10) + int(s[9]-'0')

	if month < 1 || month > 12 || day < 1 {
		return false
	}

	var maxDay int
	switch month {
	case 2:
		year := int(s[0]-'0')*1000 + int(s[1]-'0')*100 + int(s[2]-'0')*10 + int(s[3]-'0')
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			maxDay = 29
		} else {
			maxDay = 28
		}
	case 4, 6, 9, 11:
		maxDay = 30
	default:
		maxDay = 31
	}

	return day <= maxDay
}

func NewApp() *fiber.App {
	uuid.EnableRandPool()
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Println(err)
	}

	app := fiber.New(fiber.Config{
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
		AppName:     "rinha-2023-q3-romera",
	})

	// mux := http.NewServeMux()
	db := database.CreateDb()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     pkg.GetEnvOr("REDIS_HOST", "localhost") + ":" + pkg.GetEnvOr("REDIS_PORT", "6379"),
		Password: "",
		DB:       0,
		Protocol: 2,
	})

	personService := NewPersonService(db, redisClient)

	app.Post("/pessoas", func(c fiber.Ctx) error {
		var req CreatePerson
		// body, _ := io.ReadAll(r.Body)

		if err := sonic.Unmarshal(c.Body(), &req); err != nil {
			// w.WriteHeader(http.StatusBadRequest)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if req.Nickname == "" || req.Name == "" || len(req.Nickname) > 32 || len(req.Name) > 100 || !isValidDate(req.Birthday) {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		available, err := personService.ClaimNickname(req.Nickname)

		fmt.Println(available)

		if !available || err != nil {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		req.Id = uuid.NewString()
		personService.queue <- insertRequest{person: req}

		personService.CachePerson(c.Context(), req)

		c.Set("Location", fmt.Sprintf("/pessoas/%v", req.Id))
		return c.SendStatus(fiber.StatusCreated)
	})

	app.Get("/pessoas", func(c fiber.Ctx) error {
		term := c.Query("t")

		if term == "" {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		people, err := personService.GetPeopleByTerm(c.Context(), term)

		if err != nil {
			fmt.Println(err)
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		return c.JSON(people)
	})

	app.Get("/pessoas/:id", func(c fiber.Ctx) error {
		id := c.Params("id")

		val, err := redisClient.Get(c.Context(), "person:"+id).Bytes()

		if err == nil {
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Send(val)
		}

		person, err := personService.GetPersonById(c.Context(), id)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.SendStatus(fiber.StatusNotFound)
			}
			fmt.Println(err)

			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		return c.JSON(person)
	})

	app.Get("/contagem-pessoas", func(c fiber.Ctx) error {
		total := personService.Count(c.Context())

		return c.SendString(strconv.Itoa(total))
	})

	return app
}

func run() {
	app := NewApp()

	port := pkg.GetEnvOr("PORT", "80")

	fmt.Printf("running on port :%s\n", port)

	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}

func main() {
	run()
}
