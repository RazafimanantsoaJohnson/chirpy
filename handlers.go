package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
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
	Id           string    `json:"id"`
	Email        string    `json:"email"`
	Created_at   time.Time `json:"created_at"`
	Updated_at   time.Time `json:"updated_at"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

type polkaWebhookBody struct {
	Event string `json:"event"`
	Data  struct {
		UserId string `json:"user_id"`
	} `json:"data"`
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handlePostChirp(w http.ResponseWriter, r *http.Request, cfg *ApiConfig, currentUserId uuid.UUID) {
	header := w.Header()
	parameters := unmarshalRequestBody[chirp](w, r)
	parameters.UserId = currentUserId.String()
	if len(parameters.Body) > 140 {
		w.WriteHeader(400)
	} else {
		header.Add("Content-Type", "application/json")
		handleProfane(w, r, *parameters, cfg)
		return
	}
}

func handleProfane(w http.ResponseWriter, r *http.Request, reqBody chirp, cfg *ApiConfig) {
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

func (cfg *ApiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
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
		Id:          response.ID.String(),
		Email:       response.Email,
		Created_at:  response.CreatedAt,
		Updated_at:  response.UpdatedAt,
		IsChirpyRed: response.IsChirpyRed.Bool,
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

func (cfg *ApiConfig) handleListChirps(w http.ResponseWriter, r *http.Request) {
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

func (cfg *ApiConfig) handleGetChirpById(w http.ResponseWriter, r *http.Request) {
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

func (cfg *ApiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	type loginParams struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		ExpiresIn int    `json:"expires_in_seconds"`
	}
	reqBody := unmarshalRequestBody[loginParams](w, r)
	queriedUser, err := cfg.dbQueries.GetUserByEmail(r.Context(), reqBody.Email)
	err = auth.CheckPasswordHash(reqBody.Password, queriedUser.HashedPassword)
	if err != nil {
		log.Printf("error when authenticating the user: %v", err)
		w.WriteHeader(401)
		return
	}
	token, err := auth.MakeJWT(queriedUser.ID, cfg.secretKey, 1*time.Hour)
	if err != nil {
		log.Printf("error generating the JWT: %v", err)
		w.WriteHeader(500)
		return
	}

	refreshToken, err := createRefreshToken(queriedUser.ID, r, cfg)
	if err != nil {
		log.Printf("error when creating refresh token: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("error when creating refresh token"))
		return
	}

	response, err := json.Marshal(userResponse{
		Id:           queriedUser.ID.String(),
		Email:        queriedUser.Email,
		Created_at:   queriedUser.CreatedAt,
		Updated_at:   queriedUser.UpdatedAt,
		IsChirpyRed:  queriedUser.IsChirpyRed.Bool,
		Token:        token,
		RefreshToken: refreshToken,
	})
	if err != nil {
		log.Printf("error parsing the response to JSON: %v", err)
		w.WriteHeader(401)
		return
	}
	w.WriteHeader(200)
	w.Write(response)
}

func (cfg *ApiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	numHits := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	header := w.Header()
	header.Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(numHits))
}

func (cfg *ApiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	platform := os.Getenv("PLATFORM")
	if platform != "dev" {
		w.WriteHeader(403)
		return
	}
	err := cfg.dbQueries.DeleteAllUsers(r.Context())
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	cfg.fileserverHits = atomic.Int32{}
	w.WriteHeader(200)
}

func (cfg *ApiConfig) handlerAdminMetrics(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	result := fmt.Sprintf(`
		<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		</html>
	`, cfg.fileserverHits.Load())
	header.Add("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte(result))
}

func (cfg *ApiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	type refreshTokenResponse struct {
		Token string `json:"token"`
	}
	header := w.Header()
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("error happened when getting the bearer token: %v", err)
		w.WriteHeader(401)
		return
	}
	queriedRefreshToken, err := cfg.dbQueries.GetRefreshTokenById(r.Context(), bearerToken)
	if err != nil {
		log.Printf("error when querrying the refresh token %v", err)
		w.WriteHeader(401)
		return
	}
	fmt.Println(queriedRefreshToken)
	newToken, err := auth.MakeJWT(queriedRefreshToken.UserID, cfg.secretKey, 1*time.Hour)
	if err != nil {
		log.Printf("error when creating a token from the refresh token: %v", err)
		w.WriteHeader(500)
		return
	}

	jsonResponse, err := json.Marshal(&refreshTokenResponse{Token: newToken})
	if err != nil {
		log.Printf("error when parsing the new token to JSON: %v", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	header.Add("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func (cfg *ApiConfig) handlerRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	providedToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("error when getting bearer token: %v", err)
		w.WriteHeader(401)
		return
	}

	err = cfg.dbQueries.RevokeToken(r.Context(), providedToken)
	if err != nil {
		log.Printf("error when revoking token: %v", err)
		w.WriteHeader(401)
		return
	}
	w.WriteHeader(204)
}

func handlerEditUser(w http.ResponseWriter, r *http.Request, cfg *ApiConfig, curUserId uuid.UUID) {
	type editUserResponse struct {
		Id          string    `json:"id"`
		Email       string    `json:"email"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	parameters := unmarshalRequestBody[userParam](w, r)
	header := w.Header()
	logedInUser, err := cfg.dbQueries.GetUserById(r.Context(), curUserId)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	if logedInUser.Email != parameters.Email {
		log.Printf("loged in user '%v', user to update '%v'", logedInUser.Email, parameters.Email)
		header.Add("Content-Type", "text/plain")
		w.WriteHeader(401)
		w.Write([]byte("this user is not authorized to do the action"))
		return
	}

	hashedPassword, err := auth.HashPassword(parameters.Password)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	editedUser, err := cfg.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:             logedInUser.ID,
		Email:          parameters.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		w.WriteHeader(401)
		log.Printf("user to update id: %v")
		log.Printf("error when updating the user %v", err)
		return
	}

	jsonUser, err := json.Marshal(&editUserResponse{
		Id:          editedUser.ID.String(),
		Email:       editedUser.Email,
		CreatedAt:   editedUser.CreatedAt,
		UpdatedAt:   editedUser.UpdatedAt,
		IsChirpyRed: editedUser.IsChirpyRed.Bool,
	})
	if err != nil {
		w.WriteHeader(500)
		log.Printf("error when parsing the Response to JSON")
		return
	}
	header.Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(jsonUser)
}

func (cfg *ApiConfig) handlerUpgradeUserToChirpRed(w http.ResponseWriter, r *http.Request) {
	parameters := unmarshalRequestBody[polkaWebhookBody](w, r)
	providedApiKey, err := auth.GetApiKey(r.Header)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	if providedApiKey != cfg.polkaKey {
		w.WriteHeader(401)
		return
	}
	if parameters.Event == "user.upgraded" {
		userId, err := uuid.Parse(parameters.Data.UserId)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		err = cfg.dbQueries.UpgradeToChirpyRed(r.Context(), userId)
		if err != nil {
			log.Printf("error when upgrading user to chirpy red : %v", err)
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(204)
		return
	}
	w.WriteHeader(204)
}

func createRefreshToken(userId uuid.UUID, r *http.Request, cfg *ApiConfig) (string, error) {
	newRefreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		return "", err
	}
	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     newRefreshToken,
		ExpiresAt: time.Now().Add((60 * 24) * time.Hour),
		UserID:    userId,
	})
	if err != nil {
		return "", err
	}
	return newRefreshToken, nil
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
