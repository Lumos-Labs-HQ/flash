package common

import "github.com/gofiber/fiber/v2"

// JSON sends a success response with data
func JSON(c *fiber.Ctx, data any) error {
	return c.JSON(Response{Success: true, Data: data})
}

// JSONMessage sends a success response with message
func JSONMessage(c *fiber.Ctx, message string) error {
	return c.JSON(Response{Success: true, Message: message})
}

// JSONError sends an error response
func JSONError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(Response{Success: false, Message: message})
}

// JSONFiberMap sends a fiber.Map response
func JSONFiberMap(c *fiber.Ctx, data fiber.Map) error {
	return c.JSON(data)
}
