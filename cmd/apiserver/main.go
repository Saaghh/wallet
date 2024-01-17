package main

import (
	"Saaghh/wallet/internal/apiserver"
	"log"
)

// func timeHandler(w http.ResponseWriter, r *http.Request) {
//
// }

func main() {
	// r.Get("/time", timeHandler)

	// http.ListenAndServe(":8080", r)

	s := apiserver.New(apiserver.NewConfig())

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
