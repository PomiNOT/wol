package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mdlayher/wol"
)

type MachineInfoBody struct {
	Mac string
}

func IndexPage(c *fiber.Ctx) error {
	time := time.Now().Unix()

	return c.JSON(fiber.Map{
		"up":        true,
		"timestamp": time,
		"app_name":  c.App().Server().Name,
		"message":   fmt.Sprintf("WOL backend is up! Server's UNIX time is: %d", time),
	})
}

func WakeOnLan(c *fiber.Ctx) error {
	ifName, ifSet := os.LookupEnv("IFACE")

	if !ifSet {
		return fiber.NewError(
			fiber.ErrInternalServerError.Code,
			"IFACE name is not set, please set this environment variable",
		)
	}

	machineInfoBody := MachineInfoBody{}

	if err := c.BodyParser(&machineInfoBody); err != nil {
		return err
	}

	machineInfo, err := NewMachineInfoFromBody(machineInfoBody)
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "MAC address is not valid") }

	ifaceInfo, err := GetInterfaceInfo(ifName)
	if err != nil { return err }

	client, err := wol.NewClient()
	if err != nil { return err }

	err = client.Wake(
		fmt.Sprintf("%s:9", ifaceInfo.Broadcast.String()),
		machineInfo.Mac,
	)
	if err != nil { return err }

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Sending wake-up message for %s to %s", machineInfo.Mac, ifaceInfo.Broadcast),
	})
}

func DiscoverMachines(c *fiber.Ctx) error {
	ifName, ifSet := os.LookupEnv("IFACE")

	if !ifSet {
		return fiber.NewError(
			fiber.ErrInternalServerError.Code,
			"IFACE name is not set, please set this environment variable",
		)
	}

	machines, err := ARPScan(ifName)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"count": len(machines),
		"machines": machines,
	})
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"message": err.Error(),
	})
}
