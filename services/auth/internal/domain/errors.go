// Package domain — auth-specific domain errors.
//
// Why define errors here?
// The service layer returns these errors. The handler layer catches them
// and maps them to HTTP status codes using pkg/errors.IsAppError().
// The service never knows which HTTP code it maps to — that's the handler's job.
package domain

import "retail-platform/pkg/errors"

// Auth service domain errors.
// Each wraps an AppError from pkg/errors with an auth-specific code and message.
var (
	// ErrInvalidCredentials is returned for BOTH wrong email AND wrong password.
	//
	// Security reason: if you return "email not found" for wrong email and
	// "wrong password" for wrong password, attackers can probe your database
	// to discover which emails are registered. One generic error closes that
	// attack vector — the attacker learns nothing about whether the email exists.
	ErrInvalidCredentials = errors.NewUnauthorized("invalid email or password")

	// ErrEmailTaken is returned when registering with an already-used email.
	// This is a 409 Conflict — not a 400 Bad Request — because the request
	// is valid but conflicts with existing data.
	ErrEmailTaken = &errors.AppError{
		Code:    errors.ErrCodeConflict,
		Message: "an account with this email already exists",
	}

	// ErrInvalidToken is returned when a JWT or refresh token cannot be validated.
	ErrInvalidToken = &errors.AppError{
		Code:    errors.ErrCodeInvalidToken,
		Message: "invalid or malformed token",
	}

	// ErrTokenExpired is returned when a valid token has passed its expiry time.
	ErrTokenExpired = &errors.AppError{
		Code:    errors.ErrCodeUnauthorized,
		Message: "token has expired",
	}
)
