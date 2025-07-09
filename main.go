package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/RazafimanantsoaJohnson/chirpy/internal/auth"
	"github.com/RazafimanantsoaJohnson/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ApiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	secretKey      string
}

func (cfg *ApiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	result := func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("we add a value")
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(result)
}

func (cfg *ApiConfig) middlewareCheckAuth(next func(http.ResponseWriter, *http.Request, *ApiConfig, uuid.UUID)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		receivedToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(401)
			w.Write([]byte("This user is not authorized to make this request"))
			return
		}
		currentUserId, err := auth.ValidateJWT(receivedToken, cfg.secretKey)
		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(401)
			w.Write([]byte("This user is not authorized to make this request"))
			return
		}
		next(w, r, cfg, currentUserId)
	}
}

func main() {
	port := "8080"
	err := godotenv.Load()
	if err != nil {
		fmt.Errorf("server unable to read the environment variable")
		os.Exit(1)
	}
	dbUrl := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		fmt.Errorf("server unable to connect to database")
		os.Exit(1)
	}
	dbQueries := database.New(db)
	config := ApiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
		secretKey:      os.Getenv("SECRET"),
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/api/healthz", handleReadiness)
	serveMux.HandleFunc("/api/metrics", config.handlerMetrics)
	serveMux.HandleFunc("POST /api/chirps", config.middlewareCheckAuth(handlePostChirp))
	serveMux.HandleFunc("GET /api/chirps", config.handleListChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpId}", config.handleGetChirpById)
	serveMux.HandleFunc("POST /api/users", config.handleCreateUser)
	serveMux.HandleFunc("POST /api/login", config.handleLogin)
	serveMux.HandleFunc("POST /api/refresh", config.handlerRefreshToken)
	serveMux.HandleFunc("/admin/reset", config.handlerReset) // adding a namespace "admin" (in backend server means a prefix to a path)
	serveMux.HandleFunc("/admin/metrics", config.handlerAdminMetrics)
	serveMux.HandleFunc("GET /healthz", handleReadiness)
	serveMux.HandleFunc("GET /metrics", config.handlerMetrics)
	serveMux.HandleFunc("POST /reset", config.handlerReset)
	serveMux.Handle("/app/", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	server := http.Server{
		Addr:    ":" + port,
		Handler: serveMux,
	}
	fmt.Println("Server is running on port : ", port)
	server.ListenAndServe()
}
