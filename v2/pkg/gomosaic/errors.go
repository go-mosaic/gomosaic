package gomosaic

import (
	"go/token"
)

// FailedError представляет критическую ошибку с позицией в коде.
type FailedError struct {
	text    string
	posInfo *PosInfo
}

func (e *FailedError) Error() string {
	if !e.posInfo.IsValid {
		return e.text
	}
	return e.posInfo.String() + ": " + e.text
}

// WarningError представляет предупреждение с позицией в коде.
type WarningError struct {
	text string
	pos  token.Position
}

func (e *WarningError) Error() string {
	if !e.pos.IsValid() {
		return e.text
	}
	return e.pos.String() + ": " + e.text
}

// Error создает новую критическую ошибку.
func Error(text string, posInfo *PosInfo) error {
	return &FailedError{
		text:    text,
		posInfo: posInfo,
	}
}

// Warn создает новое предупреждение.
func Warn(text string, position token.Position) error {
	return &WarningError{
		text: text,
		pos:  position,
	}
}

// IsErrFailed проверяет, является ли ошибка критической.
func IsErrFailed(e error) bool {
	_, ok := e.(*FailedError)
	return ok
}

// IsErrWarning проверяет, является ли ошибка предупреждением.
func IsErrWarning(e error) bool {
	_, ok := e.(*WarningError)
	return ok
}
