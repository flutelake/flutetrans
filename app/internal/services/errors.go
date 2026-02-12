package services

import "encoding/json"

const (
	ErrCodeValidation = 1001
	ErrCodeStorage    = 1002
	ErrCodeCrypto     = 1003
	ErrCodeProtocol   = 2001
	ErrCodeAuth       = 2002
	ErrCodeTimeout    = 2003
)

type ServiceError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e ServiceError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		return e.Message
	}
	return string(b)
}

func newServiceError(code int, message string, details map[string]any) error {
	return ServiceError{Code: code, Message: message, Details: details}
}

func validationError(message string, details map[string]any) error {
	return newServiceError(ErrCodeValidation, message, details)
}

func storageError(message string, details map[string]any) error {
	return newServiceError(ErrCodeStorage, message, details)
}

func cryptoError(message string, details map[string]any) error {
	return newServiceError(ErrCodeCrypto, message, details)
}

