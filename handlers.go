package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/RazafimanantsoaJohnson/chirpy/internal/auth"
	"github.com/RazafimanantsoaJohnson/chirpy/internal/database"
	"github.com/google/uuid"
)

type chirp struct {
	Body   string `json:"body"`
	UserId string `json:"user_id"`
}

type userParam struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type chirpResponse struct {
	Id        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserId    string    `json:"user_id"`
}

type userResponse struct {
	Id         string    `json:"id"`
	Email      string    `json:"email"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlePostChirp(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	parameters := unmarshalRequestBody[chirp](w, r)

	if len(parameters.Body) > 140 {
		w.WriteHeader(400)
	} else {
		header.Add("Content-Type", "application/json")
		handleProfane(w, r, *parameters, cfg)
		return
	}
}

func handleProfane(w http.ResponseWriter, r *http.Request, reqBody chirp, cfg *apiConfig) {
	header := w.Header()
	type returnCleanedBody struct {
		CleanBody string `json:"cleaned_body"`
	}
	splittedBody := strings.Split(reqBody.Body, " ")
	for i, word := range splittedBody {
		uWord := strings.ToUpper(word)
		if uWord == "FORNAX" || uWord == "SHARBERT" || uWord == "KERFUFFLE" {
			splittedBody[i] = "****"
		}
	}
	cleanBody := returnCleanedBody{
		CleanBody: strings.Join(splittedBody, " "),
	}
	userId, err := uuid.Parse(reqBody.UserId)
	if err != nil {
		w.WriteHeader(400)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte("Provided user_id is not valid"))
		return
	}
	createdChirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanBody.CleanBody,
		UserID: userId,
	})
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte("Server Unable to insert chirp in DB"))
		return
	}
	jsonResBody, err := json.Marshal(&chirpResponse{
		Id:        createdChirp.ID.String(),
		Body:      createdChirp.Body,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		UserId:    createdChirp.UserID.String(),
	})
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte("Server unable to parse response into JSON"))
		return
	}
	w.WriteHeader(201)
	w.Write(jsonResBody)
}

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	reqBody := unmarshalRequestBody[userParam](w, r)
	hashed_password, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		log.Printf("Unable to hash password %v", err.Error())
		w.WriteHeader(500)
		return
	}
	response, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          reqBody.Email,
		HashedPassword: hashed_password,
	}) // we pass the request's context so that our db request stops (or timeouts) with the cancellation of the http request if it does
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte(err.Error()))
		return
	}
	uResponse := userResponse{
		Id:         response.ID.String(),
		Email:      response.Email,
		Created_at: response.CreatedAt,
		Updated_at: response.UpdatedAt,
	}
	jsonResponse, err := json.Marshal(&uResponse)
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(201)
	header.Add("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func (cfg *apiConfig) handleListChirps(w http.ResponseWriter, r *http.Request) {
	chirpList, err := cfg.dbQueries.GetAllChirps(r.Context())
	header := w.Header()
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte("server unable to list chirps"))
	}
	chirps := make([]chirpResponse, len(chirpList))
	for i, chirp := range chirpList {
		chirps[i] = chirpResponse{
			Id:        chirp.ID.String(),
			Body:      chirp.Body,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			UserId:    chirp.UserID.String(),
		}
	}
	jsonChirps, err := json.Marshal(&chirps)
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte("server unable to parse chirps into JSON"))
	}
	header.Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(jsonChirps)
}

func (cfg *apiConfig) handleGetChirpById(w http.ResponseWriter, r *http.Request) {
	pathChirpId := r.PathValue("chirpId")
	header := w.Header()
	chirpId, err := uuid.Parse(pathChirpId)
	if err != nil {
		header.Add("Content-Type", "text/plain")
		w.WriteHeader(404)
		w.Write([]byte("provided chirpId non-valid"))
		return
	}
	chirp, err := cfg.dbQueries.GetChirpById(r.Context(), chirpId)
	if err != nil {
		header.Add("Content-Type", "text/plain")
		w.WriteHeader(404)
		w.Write([]byte("provided chirpId non-valid"))
		return
	}
	response, err := json.Marshal(&chirpResponse{
		Id:        chirp.ID.String(),
		Body:      chirp.Body,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		UserId:    chirp.UserID.String(),
	})
	header.Add("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write(response)
}

func (cfg *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	type loginParams struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	reqBody := unmarshalRequestBody[loginParams](w, r)
	queriedUser, err := cfg.dbQueries.GetUserByEmail(r.Context(), reqBody.Email)
	err = auth.CheckPasswordHash(reqBody.Password, queriedUser.HashedPassword)
	if err != nil {
		log.Printf("error when authenticating the user: %v", err)
		w.WriteHeader(401)
		return
	}
	response, err := json.Marshal(userResponse{
		Id:         queriedUser.ID.String(),
		Email:      queriedUser.Email,
		Created_at: queriedUser.CreatedAt,
		Updated_at: queriedUser.UpdatedAt,
	})
	if err != nil {
		log.Printf("error parsing the response to JSON: %v", err)
		w.WriteHeader(401)
		return
	}
	w.WriteHeader(200)
	w.Write(response)
}

func unmarshalRequestBody[T any](w http.ResponseWriter, r *http.Request) *T { // using generics is the way to go for functions to handle many types
	var unmarshalledReqBody T
	reqBody, err := io.ReadAll(r.Body)
	err = json.Unmarshal(reqBody, &unmarshalledReqBody)
	header := w.Header()
	if err != nil {
		header.Add("Content-Type", "text/plain")
		w.WriteHeader(400)
		w.Write([]byte("Server unable to read request body"))
	}
	return &unmarshalledReqBody
}
