package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/RazafimanantsoaJohnson/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	result := func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("we add a value")
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(result)
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	numHits := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	header := w.Header()
	header.Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(numHits))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
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

func (cfg *apiConfig) handlerAdminMetrics(w http.ResponseWriter, r *http.Request) {
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
	config := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/api/healthz", handleReadiness)
	serveMux.HandleFunc("/api/metrics", config.handlerMetrics)
	serveMux.HandleFunc("POST /api/chirps", config.handlePostChirp)
	serveMux.HandleFunc("GET /api/chirps", config.handleListChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpId}", config.handleGetChirpById)
	serveMux.HandleFunc("POST /api/users", config.handleCreateUser)
	serveMux.HandleFunc("POST /api/login", config.handleLogin)
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
