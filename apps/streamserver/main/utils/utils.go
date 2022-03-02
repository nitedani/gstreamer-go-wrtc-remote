package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type ParseJsonValue[T any] struct {
	Value T
	Error error
}

func ParseJson[T any](response *resty.Response) ParseJsonValue[T] {
	parsed := ParseJsonValue[T]{}
	err := json.Unmarshal(response.Body(), &parsed.Value)

	if err != nil {
		fmt.Println(err)
		return ParseJsonValue[T]{Error: err}
	}
	return parsed
}

func RandomStr() string {
	n := 5
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
