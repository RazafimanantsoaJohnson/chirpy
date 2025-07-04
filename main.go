package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	config := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/api/healthz", handleReadiness)
	serveMux.HandleFunc("/api/metrics", config.handlerMetrics)
	serveMux.HandleFunc("/api/reset", config.handlerReset)
	serveMux.HandleFunc("/admin/reset", config.handlerReset) // adding a namespace "admin" (in backend server means a prefix to a path)
	serveMux.HandleFunc("/admin/metrics", config.handlerAdminMetrics)
	serveMux.Handle("/app/", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	server := http.Server{
		Addr:    ":" + port,
		Handler: serveMux,
	}
	fmt.Println("Server is running on port : ", port)
	server.ListenAndServe()
}
