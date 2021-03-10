package checkly

import (
	"os"
	"strconv"
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
