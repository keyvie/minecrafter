package validator

import (
	val "github.com/go-playground/validator/v10"
)

type Validator struct {
	V *val.Validate
}

func New() *Validator {
	v := val.New(val.WithRequiredStructEnabled())
	if err := v.RegisterValidation("snowflake", func(fl val.FieldLevel) bool {
		s := fl.Field().String()
		if len(s) < 17 || len(s) > 20 {
			return false
		}
		return true
	}); err != nil {
		panic(err)
	}
	return &Validator{V: v}
}
