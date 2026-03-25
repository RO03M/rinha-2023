package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"rinha/database"
	"rinha/pkg"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
	})
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
	return true
}

func NewHandler() http.Handler {
	uuid.EnableRandPool()
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Println(err)
	}

	mux := http.NewServeMux()
	db := database.CreateDb()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     pkg.GetEnvOr("REDIS_HOST", "localhost") + ":" + pkg.GetEnvOr("REDIS_PORT", "6379"),
		Password: "",
		DB:       0,
		Protocol: 2,
	})

	personService := NewPersonService(db, redisClient)

	mux.HandleFunc("POST /pessoas", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req CreatePerson
		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Body inválido")
			return
		}

		if len(req.Nickname) > 32 || len(req.Name) > 100 || !isValidDate(req.Birthday) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		available, err := personService.ClaimNickname(req.Nickname)

		if !available || err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		// id, err := personService.CreatePerson(r.Context(), req.Name, req.Nickname, req.Birthday, req.Stack)
		req.Id = uuid.NewString()
		personService.queue <- insertRequest{
			person: req,
		}

		personService.CachePerson(r.Context(), req)

		// if err != nil {
		// 	fmt.Println(err)
		// 	w.WriteHeader(http.StatusUnprocessableEntity)
		// 	return
		// }

		w.Header().Set("Location", fmt.Sprintf("/pessoas/%v", req.Id))
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /pessoas", func(w http.ResponseWriter, r *http.Request) {
		term := r.URL.Query().Get("t")

		if term == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		people, err := personService.GetPeopleByTerm(r.Context(), term)

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.Encode(people)
	})

	mux.HandleFunc("GET /pessoas/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		val, err := redisClient.Get(r.Context(), "person:"+id).Bytes()

		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(val)
			return
		}

		person, err := personService.GetPersonById(r.Context(), id)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			fmt.Println(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(person)
	})
	mux.HandleFunc("GET /contagem-pessoas", func(w http.ResponseWriter, r *http.Request) {
		total := personService.Count(r.Context())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strconv.Itoa(total)))
	})

	return mux
}

func run() {
	handler := NewHandler()

	port := pkg.GetEnvOr("PORT", "80")

	server := http.Server{
		Addr:     fmt.Sprintf(":%s", port),
		Handler:  handler,
		ErrorLog: log.New(os.Stderr, "http: ", log.LstdFlags),
	}

	fmt.Printf("running on port :%s\n", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	run()
}
