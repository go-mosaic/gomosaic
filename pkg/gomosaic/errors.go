package gomosaic

import (
	"go/token"
)

type Level string

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

func Error(text string, posInfo *PosInfo) error {
	return &FailedError{
		text:    text,
		posInfo: posInfo,
	}
}

func Warn(text string, position token.Position) error {
	return &WarningError{
		text: text,
		pos:  position,
	}
}

func IsErrFailed(e error) bool {
	_, ok := e.(*FailedError)
	return ok
}

func IsErrWarning(e error) bool {
	_, ok := e.(*WarningError)
	return ok
}
