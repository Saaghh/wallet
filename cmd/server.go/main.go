package main

import (
	"fmt"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Write([]byte(time.Now().String()))
		return
	default:
		http.Error(w, "Not Allowed Method", http.StatusNotAcceptable)
		return
	}
}

func main() {
	http.HandleFunc("/time", handler)

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
	}
}
