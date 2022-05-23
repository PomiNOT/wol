package main

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

type MachineInfo struct {
	Mac string
}

func IndexPage(c *fiber.Ctx) error {
	time := time.Now().Unix()
	return c.JSON(fiber.Map {
		"up": true,
		"timestamp": time,
		"app_name": c.App().Server().Name,
		"message": fmt.Sprintf("WOL backend is up! Server's UNIX time is: %d", time),
	})
}

func WakeOnLan(c *fiber.Ctx) error {
	machineInfo := MachineInfo{}
	err := c.BodyParser(&machineInfo)
	if err != nil {
		return c.Status(400).JSON(fiber.Map {
			"message": "Bad request!",
		})
	}

	return c.JSON(fiber.Map {
		"message": fmt.Sprintf("Waking up %s", machineInfo.Mac),
	})
}

func NotFound(c *fiber.Ctx) error {
	return c.Status(404).JSON(fiber.Map {
		"message": "Route not found.",
	})
}