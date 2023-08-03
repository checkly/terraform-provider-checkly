package checkly

import (
	"encoding/json"
	"fmt"
)

// ErrorLog defines ErrorLog type
type ErrorLog map[string]interface{}

// ErrorWithLog defines checkly error type
type ErrorWithLog struct {
	Err  string
	Data *ErrorLog
}

func (e ErrorWithLog) Error() string {
	data, _ := json.Marshal(e.Data)
	return fmt.Sprintf("%s [%s]", e.Err, data)
}

func makeError(err string, l *ErrorLog) ErrorWithLog {
	return ErrorWithLog{err, l}
}
