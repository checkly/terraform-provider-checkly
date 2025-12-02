package checkly

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func apiCallTimeout() time.Duration {
	timeout := os.Getenv("API_CALL_TIMEOUT")
	if timeout != "" {
		v, err := strconv.ParseInt(timeout, 10, 64)
		if err != nil || v < 1 {
			panic("Invalid API_CALL_TIMEOUT value, must be a positive number")
		} else {
			return time.Duration(v) * time.Second
		}
	}
	return 15 * time.Second
}

func checksumSha256(r io.Reader) string {
	hash := sha256.New()

	_, err := io.Copy(hash, r)
	if err != nil {
		panic("failed to calculate checksum: " + err.Error())
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	return checksum
}

type allowedValue[T any] struct {
	Value       T
	Description string
}

func (v *allowedValue[T]) String() string {
	if v.Description != "" {
		return fmt.Sprintf("`%v` (%s)", v.Value, v.Description)
	}

	return fmt.Sprintf("`%v`", v.Value)
}

type allowedValues[T any] []allowedValue[T]

func (v *allowedValues[T]) Values() []T {
	s := make([]T, 0, len(*v))

	for _, value := range *v {
		s = append(s, value.Value)
	}

	return s
}

func (v *allowedValues[T]) String() string {
	l := len(*v)
	switch l {
	case 0:
		return "There are no allowed values."
	case 1:
		return fmt.Sprintf("The only allowed value is %s.", (*v)[0].String())
	default:
		head := (*v)[:l-1]
		last := (*v)[l-1]

		var buf strings.Builder

		buf.WriteString("The allowed values are ")

		for i, value := range head {
			if i > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(value.String())
		}

		buf.WriteString(" and ")
		buf.WriteString(last.String())
		buf.WriteString(".")

		return buf.String()
	}
}
