package handler

import (
	"github.com/gofiber/fiber/v2"
	"wzap/internal/model"
	"wzap/internal/service"
)

type UserHandler struct {
	userSvc *service.UserService
}

func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// Create godoc
// @Summary     Create a new user (Admin Only)
// @Description Creates a new user with an auto-generated token and associated session
// @Tags        Users
// @Accept      json
// @Produce     json
// @Param       body body     model.UserCreateReq true "User data"
// @Success     200  {object} model.APIResponse
// @Failure     400  {object} model.APIError
// @Security    BearerAuth
// @Router      /users [post]
func (h *UserHandler) Create(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Admin access required"))
	}

	var req model.UserCreateReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", err.Error()))
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(model.ErrorResp("Bad Request", "name is required"))
	}

	user, err := h.userSvc.Create(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(user, "User created successfully"))
}

// List godoc
// @Summary     List users (Admin Only)
// @Description Returns all users
// @Tags        Users
// @Produce     json
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /users [get]
func (h *UserHandler) List(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Admin access required"))
	}

	users, err := h.userSvc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(users, "Users retrieved"))
}

// Get godoc
// @Summary     Get user by ID (Admin Only)
// @Description Returns a specific user
// @Tags        Users
// @Produce     json
// @Param       id path string true "User ID"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /users/{id} [get]
func (h *UserHandler) Get(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Admin access required"))
	}

	id := c.Params("id")
	user, err := h.userSvc.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.ErrorResp("Not Found", err.Error()))
	}

	return c.JSON(model.SuccessResp(user, "User retrieved"))
}

// Delete godoc
// @Summary     Delete user (Admin Only)
// @Description Deletes a user and its associated sessions/webhooks
// @Tags        Users
// @Produce     json
// @Param       id path string true "User ID"
// @Success     200 {object} model.APIResponse
// @Security    BearerAuth
// @Router      /users/{id} [delete]
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	if c.Locals("authRole") != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(model.ErrorResp("Forbidden", "Admin access required"))
	}

	id := c.Params("id")
	if err := h.userSvc.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.ErrorResp("Internal Server Error", err.Error()))
	}

	return c.JSON(model.SuccessResp(nil, "User deleted"))
}
