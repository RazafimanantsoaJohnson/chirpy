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

func main() {
	port := "8080"
	config := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/healthz", handleReadiness)
	serveMux.HandleFunc("/metrics", config.handlerMetrics)
	serveMux.HandleFunc("/reset", config.handlerReset)
	serveMux.Handle("/app/", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	server := http.Server{
		Addr:    ":" + port,
		Handler: serveMux,
	}
	fmt.Println("Server is running on port : ", port)
	server.ListenAndServe()
}
