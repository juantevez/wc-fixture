package testutil

import (
	"errors"
	"testing"

	"github.com/wc-fixture/shared/pkg/apperrors"
)

// AssertNoError falla el test si err no es nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("se esperaba nil, se obtuvo error: %v", err)
	}
}

// AssertError falla el test si err es nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("se esperaba un error, se obtuvo nil")
	}
}

// AssertAppError falla el test si err no es un *apperrors.AppError con el código dado.
//
//	testutil.AssertAppError(t, err, apperrors.CodeNotFound)
func AssertAppError(t *testing.T, err error, code apperrors.ErrCode) {
	t.Helper()
	var ae *apperrors.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("se esperaba *apperrors.AppError, se obtuvo %T: %v", err, err)
	}
	if ae.Code != code {
		t.Fatalf("se esperaba código %q, se obtuvo %q (mensaje: %s)", code, ae.Code, ae.Message)
	}
}

// AssertEqual falla el test si got != want.
func AssertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if want != got {
		t.Fatalf("AssertEqual: want %v, got %v", want, got)
	}
}

// AssertLen falla el test si len(slice) != expectedLen.
func AssertLen[T any](t *testing.T, slice []T, expectedLen int) {
	t.Helper()
	if len(slice) != expectedLen {
		t.Fatalf("AssertLen: se esperaba longitud %d, se obtuvo %d", expectedLen, len(slice))
	}
}

// AssertNotNil falla el test si v es nil.
func AssertNotNil(t *testing.T, v any) {
	t.Helper()
	if v == nil {
		t.Fatal("AssertNotNil: se esperaba un valor no-nil")
	}
}
