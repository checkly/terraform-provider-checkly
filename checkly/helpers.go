package checkly

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
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

func checksumSha256(r io.Reader) string {
	hash := sha256.New()

	_, err := io.Copy(hash, r)
	if err != nil {
		panic("failed to calculate checksum: " + err.Error())
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	return checksum
}
