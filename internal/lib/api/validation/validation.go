package val

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func ValidationError(errs validator.ValidationErrors) string {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "username":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not a valid username", err.Field()))
		case "password":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not a valid password", err.Field()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not valid", err.Field()))
		}
	}

	errText := strings.Join(errMsgs, ", ")

	return errText
}