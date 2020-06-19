package main

import (
	"fmt"
	"log"
	"net/http"
)

func FinalHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {

	mux := http.NewServeMux()

	limitHandler := http.HandlerFunc(FinalHandler)
	mux.Handle("/", rateLimiterMiddleWare(limitHandler))

	log.Println("Listening on port : 4000...")
	err := http.ListenAndServe(":4000", mux)
	if err != nil {
		fmt.Println("", err.Error())
	}
}