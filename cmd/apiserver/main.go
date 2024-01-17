package main

import (
	"Saaghh/wallet/internal/apiserver"
	"log"
)

// func timeHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method == http.MethodGet {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte(time.Now().String()))

// 		return
// 	}
// }

func main() {
	// r := chi.NewRouter()
	// r.Get("/time", timeHandler)

	// http.ListenAndServe(":8080", r)

	s := apiserver.New(apiserver.NewConfig())

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
