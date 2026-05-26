package checkly

import (
	"cmp"
	"fmt"
	"os"
	"regexp"
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

func validateFileExists() func(val any, key string) (warns []string, errs []error) {
	return func(val any, key string) (warns []string, errs []error) {
		v := val.(string)

		_, err := os.Stat(v)
		if os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("%q refers to a non-existing file %q: %w", key, v, err))
		}

		return warns, errs
	}
}

// validateGzipArchive checks that the file at the given path is a gzip archive
// by inspecting the first two bytes (magic number 0x1f 0x8b). If the file
// appears to be a zip archive instead, the error message says so.
func validateGzipArchive() func(val any, key string) (warns []string, errs []error) {
	return func(val any, key string) (warns []string, errs []error) {
		v := val.(string)

		f, err := os.Open(v)
		if err != nil {
			// Let validateFileExists handle this.
			return warns, errs
		}
		defer f.Close()

		var magic [2]byte
		if _, err := f.Read(magic[:]); err != nil {
			errs = append(errs, fmt.Errorf("%q could not be read: %w", key, err))
			return warns, errs
		}

		if magic[0] == 0x50 && magic[1] == 0x4b {
			errs = append(errs, fmt.Errorf(
				"%q appears to be a .zip archive, but a .tar.gz archive is required", key,
			))
			return warns, errs
		}

		if magic[0] != 0x1f || magic[1] != 0x8b {
			errs = append(errs, fmt.Errorf(
				"%q is not a valid .tar.gz archive", key,
			))
		}

		return warns, errs
	}
}

func validateAll(validators ...func(any, string) ([]string, []error)) func(any, string) ([]string, []error) {
	return func(val any, key string) (warns []string, errs []error) {
		for _, v := range validators {
			w, e := v(val, key)
			warns = append(warns, w...)
			errs = append(errs, e...)
			if len(errs) > 0 {
				return warns, errs
			}
		}
		return warns, errs
	}
}

var versionFormatRegex = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)

func validateVersionFormat(val any, key string) (warns []string, errs []error) {
	v := val.(string)
	if !versionFormatRegex.MatchString(v) {
		errs = append(errs, fmt.Errorf("%q must be a version number (e.g. \"22\", \"24\", \"1.3\"), got: %s", key, v))
	}
	return warns, errs
}
