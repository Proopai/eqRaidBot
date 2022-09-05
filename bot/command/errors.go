package command

import "errors"

var (
	ErrorInvalidInput  = errors.New("invalid input, there was a problem with your input. Please review the choices and try again")
	ErrorInternalError = errors.New("there was a problem with this request, please try again. If the problem persists - contact your administrator")
)
