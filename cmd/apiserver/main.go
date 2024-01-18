package main

import (
	"github.com/Saaghh/wallet/internal/apiserver"
)

func main() {
	s := apiserver.New(apiserver.NewConfig())

	if err := s.Start(); err != nil {
		panic(err)
	}
}
