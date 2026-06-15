// Package notification contiene el modelo de dominio del bounded context notification.
// Su responsabilidad es consumir eventos de fixture-core y distribuirlos
// a clientes suscritos via webhooks y Server-Sent Events (SSE).
//
// notification es un Generic Subdomain: no contiene lógica de negocio del torneo,
// solo transforma eventos de dominio en notificaciones hacia el exterior.
package notification

import "fmt"

// DomainError es el error tipado del dominio de notification.
type DomainError struct {
	Code    ErrCode
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type ErrCode string

const (
	ErrCodeInvalidWebhookURL  ErrCode = "INVALID_WEBHOOK_URL"
	ErrCodeInvalidEventType   ErrCode = "INVALID_EVENT_TYPE"
	ErrCodeWebhookDeliveryFailed ErrCode = "WEBHOOK_DELIVERY_FAILED"
	ErrCodeSubscriberNotFound ErrCode = "SUBSCRIBER_NOT_FOUND"
)

func ErrInvalidWebhookURL(url string) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidWebhookURL,
		Message: fmt.Sprintf("URL de webhook inválida: %q", url),
	}
}

func ErrWebhookDeliveryFailed(url string, statusCode int) *DomainError {
	return &DomainError{
		Code:    ErrCodeWebhookDeliveryFailed,
		Message: fmt.Sprintf("entrega fallida a %q: status %d", url, statusCode),
	}
}
