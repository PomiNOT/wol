package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

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
	machineInfo := MachineInfo {}
	
	if err := c.BodyParser(&machineInfo); err != nil {
		return fiber.ErrInternalServerError
	}

	validMac, err := machineInfo.validMac()

	if err != nil {
		return fiber.ErrInternalServerError
	}

	if !validMac {
		return fiber.NewError(fiber.ErrBadRequest.Code, "MAC address is not valid")
	}

	return c.JSON(fiber.Map {
		"message": fmt.Sprintf("Waking up %s", machineInfo.Mac),
	})
}

func DiscoverMachines(c *fiber.Ctx) error {
	ifName, ifSet := os.LookupEnv("IFACE")

	if !ifSet {
		return fiber.NewError(fiber.ErrInternalServerError.Code, "IFACE name is not set, please set this environment variable")
	}

	machines, err := ARPScan(ifName)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map {
		"machines": machines,
	})
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map {
		"message": err.Error(),
	})
}