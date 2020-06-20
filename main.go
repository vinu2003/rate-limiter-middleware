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
	mData := new(middlewareData)

	limitHandler := http.HandlerFunc(FinalHandler)
	mux.Handle("/", mData.rateLimiterMiddleWare(limitHandler))

	log.Println("Listening on port : 4000...")
	err := http.ListenAndServe(":4000", mux)
	if err != nil {
		fmt.Println("", err.Error())
	}
}