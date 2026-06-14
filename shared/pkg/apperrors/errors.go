package apperrors

import "fmt"

// AppError es el error tipado de aplicación. Encapsula un código semántico,
// un mensaje legible para el cliente y opcionalmente el error interno original.
type AppError struct {
	Code    ErrCode
	Message string
	Err     error // causa interna, nunca expuesta al cliente
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// Is permite usar errors.Is comparando por código.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// ── Constructores ────────────────────────────────────────────────────────────

func NotFound(resource, id string) *AppError {
	return &AppError{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s %q no encontrado", resource, id),
	}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg}
}

func Validation(msg string) *AppError {
	return &AppError{Code: CodeValidation, Message: msg}
}

func ValidationF(format string, args ...any) *AppError {
	return &AppError{Code: CodeValidation, Message: fmt.Sprintf(format, args...)}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg}
}

func Internal(msg string, cause error) *AppError {
	return &AppError{Code: CodeInternal, Message: msg, Err: cause}
}

func Unavailable(msg string) *AppError {
	return &AppError{Code: CodeUnavailable, Message: msg}
}

// ── Sentinels para errors.Is ─────────────────────────────────────────────────

var (
	ErrNotFound     = &AppError{Code: CodeNotFound}
	ErrConflict     = &AppError{Code: CodeConflict}
	ErrValidation   = &AppError{Code: CodeValidation}
	ErrUnauthorized = &AppError{Code: CodeUnauthorized}
	ErrForbidden    = &AppError{Code: CodeForbidden}
	ErrInternal     = &AppError{Code: CodeInternal}
)

// IsCode reporta si err es un AppError con el código dado.
func IsCode(err error, code ErrCode) bool {
	var ae *AppError
	// recorre la cadena de Unwrap
	for e := err; e != nil; {
		if ae, ok := e.(*AppError); ok {
			return ae.Code == code
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			break
		}
		e = u.Unwrap()
	}
	_ = ae
	return false
}
