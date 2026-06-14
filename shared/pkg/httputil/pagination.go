package httputil

import (
	"net/http"
	"strconv"

	"github.com/wc-fixture/shared/pkg/apperrors"
)

const (
	defaultPage    = 1
	defaultPerPage = 20
	maxPerPage     = 100
)

// PageParams contiene los parámetros de paginación parseados del request.
type PageParams struct {
	Page    int
	PerPage int
}

// Offset calcula el offset para la query SQL.
func (p PageParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit retorna el límite para la query SQL (alias de PerPage para legibilidad).
func (p PageParams) Limit() int {
	return p.PerPage
}

// ParsePagination extrae y valida los query params "page" y "per_page".
// Valores por defecto: page=1, per_page=20. Máximo per_page=100.
func ParsePagination(r *http.Request) (PageParams, error) {
	params := PageParams{Page: defaultPage, PerPage: defaultPerPage}

	if v := r.URL.Query().Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return params, apperrors.Validation("el parámetro 'page' debe ser un entero mayor a 0")
		}
		params.Page = n
	}

	if v := r.URL.Query().Get("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return params, apperrors.Validation("el parámetro 'per_page' debe ser un entero mayor a 0")
		}
		if n > maxPerPage {
			return params, apperrors.ValidationF("el parámetro 'per_page' no puede superar %d", maxPerPage)
		}
		params.PerPage = n
	}

	return params, nil
}

// PageMeta es el metadata de paginación incluido en respuestas de listados.
type PageMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// NewPageMeta construye el metadata de paginación dados los parámetros y el total.
func NewPageMeta(params PageParams, totalItems int) PageMeta {
	totalPages := totalItems / params.PerPage
	if totalItems%params.PerPage != 0 {
		totalPages++
	}
	return PageMeta{
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

// PagedResponse es el envelope para respuestas paginadas.
//
//	{ "data": [...], "meta": { "page": 1, "per_page": 20, ... } }
type PagedResponse[T any] struct {
	Data []T      `json:"data"`
	Meta PageMeta `json:"meta"`
}

// WritePagedJSON escribe una respuesta paginada con status 200.
func WritePagedJSON[T any](w http.ResponseWriter, items []T, meta PageMeta) {
	WriteJSON(w, 200, PagedResponse[T]{Data: items, Meta: meta})
}
