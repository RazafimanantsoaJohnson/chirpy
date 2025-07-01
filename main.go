package main

import (
	"fmt"
	"net/http"
)

func main() {
	port := "8080"
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/healthz/", handleReadiness)
	serveMux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	server := http.Server{
		Addr:    ":" + port,
		Handler: serveMux,
	}
	fmt.Println("Server is running on port : ", port)
	server.ListenAndServe()
}
