package response

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// ErrorCode defines unified error codes.
// ErrorCode 定义统一错误码。
type ErrorCode int

const (
	CodeOK ErrorCode = 0

	CodeBadRequest   ErrorCode = 10000
	CodeUnauthorized ErrorCode = 10001
	CodeForbidden    ErrorCode = 10002
	CodeNotFound     ErrorCode = 10003
	CodeConflict     ErrorCode = 10004
	CodeInternal     ErrorCode = 10005
)

// Envelope is a unified API response wrapper.
// Envelope 是统一 API 返回结构。
type Envelope[T any] struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"` // English message / 英文信息
	Data    T         `json:"data,omitempty"`
}

// OK returns success response.
// OK 返回成功响应。
func OK[T any](c fiber.Ctx, data T) error {
	return c.Status(http.StatusOK).JSON(Envelope[T]{
		Code:    CodeOK,
		Message: "ok",
		Data:    data,
	})
}

// Fail returns error response.
// Fail 返回错误响应。
func Fail(c fiber.Ctx, httpStatus int, code ErrorCode, msg string) error {
	return c.Status(httpStatus).JSON(Envelope[any]{
		Code:    code,
		Message: msg,
	})
}

// BadRequest helper.
// BadRequest 辅助函数。
func BadRequest(c fiber.Ctx, msg string) error {
	return Fail(c, http.StatusBadRequest, CodeBadRequest, msg)
}

// Unauthorized helper.
// Unauthorized 辅助函数。
func Unauthorized(c fiber.Ctx, msg string) error {
	return Fail(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}

// Forbidden helper.
// Forbidden 辅助函数。
func Forbidden(c fiber.Ctx, msg string) error {
	return Fail(c, http.StatusForbidden, CodeForbidden, msg)
}

// NotFound helper.
// NotFound 辅助函数。
func NotFound(c fiber.Ctx, msg string) error {
	return Fail(c, http.StatusNotFound, CodeNotFound, msg)
}

// Conflict helper.
// Conflict 辅助函数。
func Conflict(c fiber.Ctx, msg string) error {
	return Fail(c, http.StatusConflict, CodeConflict, msg)
}

// Internal helper.
// Internal 辅助函数。
func Internal(c fiber.Ctx, msg string) error {
	return Fail(c, http.StatusInternalServerError, CodeInternal, msg)
}
