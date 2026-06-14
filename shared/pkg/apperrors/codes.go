package apperrors

// ErrCode identifica la categoría semántica del error de aplicación.
// Los handlers HTTP mapean cada código a un status code.
type ErrCode string

const (
	CodeNotFound     ErrCode = "NOT_FOUND"
	CodeConflict     ErrCode = "CONFLICT"
	CodeValidation   ErrCode = "VALIDATION"
	CodeUnauthorized ErrCode = "UNAUTHORIZED"
	CodeForbidden    ErrCode = "FORBIDDEN"
	CodeInternal     ErrCode = "INTERNAL"
	CodeUnavailable  ErrCode = "UNAVAILABLE"
)

// HTTPStatus retorna el status code HTTP canónico para cada código.
func (c ErrCode) HTTPStatus() int {
	switch c {
	case CodeNotFound:
		return 404
	case CodeConflict:
		return 409
	case CodeValidation:
		return 422
	case CodeUnauthorized:
		return 401
	case CodeForbidden:
		return 403
	case CodeUnavailable:
		return 503
	default:
		return 500
	}
}
