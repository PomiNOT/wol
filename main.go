package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func getPort(defaultPort int) int {
	portEnv, isSet := os.LookupEnv("PORT")
	if isSet {
		newPort, err := strconv.Atoi(portEnv)
		if err != nil {
			log.Fatal("Invalid PORT, specify correctly or leave for default (3000).")
		}
		defaultPort = newPort
	}

	return defaultPort
}

func main() {
	app := fiber.New(fiber.Config{
		AppName: "WOL",
		ServerHeader: "WOL Backend Server",
		ErrorHandler: ErrorHandler,
	})

	app.Get("/", IndexPage)
	app.Get("/discover", DiscoverMachines)
	app.Post("/wake", WakeOnLan)

	log.Fatal(app.Listen(fmt.Sprintf(":%d", getPort(3000))))
}