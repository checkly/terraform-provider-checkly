package checkly

import (
	"cmp"
	"fmt"
	"slices"
)

func validateOneOf[T comparable](allowed []T) func(val any, key string) (warns []string, errs []error) {
	return func(val any, key string) (warns []string, errs []error) {
		v := val.(T)
		if !slices.Contains(allowed, v) {
			errs = append(errs, fmt.Errorf("%q must be one of %v, got: %v", key, allowed, v))
		}
		return warns, errs
	}
}

func validateBetween[T cmp.Ordered](from, to T) func(val any, key string) (warns []string, errs []error) {
	return func(val any, key string) (warns []string, errs []error) {
		v := val.(T)
		if v < from || v > to {
			errs = append(errs, fmt.Errorf("%q must be between %v and %v, got: %v", key, from, to, v))
		}
		return warns, errs
	}
}
