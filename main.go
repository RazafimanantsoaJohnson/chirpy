package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type chirp struct {
	Body string `json:"body"`
}

type chirpResponse struct {
	Err   string `json:"error"`
	Valid bool   `json:"valid"`
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

func handlePostChirp(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(r.Body)
	var unmarshalledReqBody chirp
	var resBody chirpResponse
	err = json.Unmarshal(reqBody, &unmarshalledReqBody)
	header := w.Header()
	if err != nil {
		header.Add("Content-Type", "text/plain")
		w.WriteHeader(400)
		w.Write([]byte("Server unable to read request body"))
	}

	if len(unmarshalledReqBody.Body) > 140 {
		resBody.Err = "Something went wrong"
		w.WriteHeader(400)
	} else {
		resBody.Valid = true
		w.WriteHeader(200)
	}
	jsonResBody, err := json.Marshal(&resBody)
	if err != nil {
		w.WriteHeader(500)
		header.Add("Content-Type", "text/plain")
		w.Write([]byte("Server unable to parse response into JSON"))
	}
	w.Write(jsonResBody)

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
	serveMux.HandleFunc("POST /api/validate_chirp", handlePostChirp)
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
