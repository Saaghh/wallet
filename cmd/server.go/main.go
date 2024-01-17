package main

import (
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func timeHandler(c echo.Context) error {
	return c.String(http.StatusOK, time.Now().String())
}

func mw(next echo.HandlerFunc) echo.HandlerFunc {
	return nil
}

func main() {
	e := echo.New()

	e.Use()

	e.GET("/time", timeHandler)

	if err := e.Start(":8080"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
