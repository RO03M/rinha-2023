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

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
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

func NewHandler() http.Handler {
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Println(err)
	}

	mux := http.NewServeMux()
	db := database.CreateDb()

	personService := PersonService{
		db: db,
	}

	mux.HandleFunc("POST /pessoas", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req CreatePerson
		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Body inválido")
			return
		}

		id, err := personService.CreatePerson(r.Context(), req.Name, req.Nickname, req.Birthday, req.Stack)

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/pessoas/%v", id))
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
