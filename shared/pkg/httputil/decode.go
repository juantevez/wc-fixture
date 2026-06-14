package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/wc-fixture/shared/pkg/apperrors"
)

const maxBodyBytes = 1 << 20 // 1 MB

// DecodeJSON deserializa el cuerpo JSON del request en dst.
// Retorna un *apperrors.AppError con código Validation si:
//   - el cuerpo está vacío
//   - el JSON es malformado
//   - hay campos desconocidos (DisallowUnknownFields)
//   - el cuerpo supera 1 MB
//
// El llamador puede pasar el error directamente a WriteError.
func DecodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return mapDecodeError(err)
	}

	// Verificar que no haya un segundo objeto en el body.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperrors.Validation("el cuerpo debe contener un único objeto JSON")
	}

	return nil
}

// mapDecodeError convierte errores de json.Decoder a AppError de validación.
func mapDecodeError(err error) error {
	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxErr):
		return apperrors.ValidationF("JSON malformado en posición %d", syntaxErr.Offset)
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return apperrors.Validation("el cuerpo de la solicitud está vacío")
	case errors.As(err, &unmarshalErr):
		return apperrors.ValidationF("tipo inválido para el campo %q: se esperaba %s", unmarshalErr.Field, unmarshalErr.Type)
	default:
		// Campo desconocido u otro error
		msg := err.Error()
		if len(msg) > 120 {
			msg = msg[:120]
		}
		return apperrors.ValidationF("error al decodificar JSON: %s", fmt.Sprintf("%s", msg))
	}
}
