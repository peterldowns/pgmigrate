package shared

import (
	"fmt"
	"strings"
)

type Required interface {
	Name() string
	IsSet() bool
}

func Validate(vars ...Required) error {
	missing := []string{}
	for _, s := range vars {
		if !s.IsSet() {
			missing = append(missing, s.Name())
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if len(missing) == 1 {
		return fmt.Errorf(`required flag "%s" not set`, missing[0])
	}
	return fmt.Errorf(`required flags "%s" not set`, strings.Join(missing, `", "`))
}

func NewVariable[T comparable](name string, values ...T) Variable[T] {
	var result T // starts at zero value
	for _, v := range values {
		if v != result {
			result = v
			break
		}
	}
	return Variable[T]{name: name, value: result}
}

type Variable[T comparable] struct {
	name  string
	value T
}

func (s Variable[T]) Name() string {
	return s.name
}

func (s Variable[T]) IsSet() bool {
	var zero T
	return s.value != zero
}

func (s Variable[T]) Value() T {
	return s.value
}
