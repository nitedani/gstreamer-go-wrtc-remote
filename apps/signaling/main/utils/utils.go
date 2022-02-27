package utils

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v5"
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

func ParseBody[T any](c echo.Context) ParseJsonValue[T] {
	parsed := ParseJsonValue[T]{}
	err := c.Bind(&parsed.Value)
	if err != nil {
		fmt.Println(err)
		return ParseJsonValue[T]{Error: err}
	}
	return parsed
}

type ResolveFunc[T any] func(value T)
type RejectFunc func(err error)
type PromiseCallback[T any] func(resolve ResolveFunc[T], reject RejectFunc)
type PromiseValue[T any] struct {
	Value T
	Error error
}

type PromiseReturnType[T any] chan PromiseValue[T]

func Promise[T any](cb PromiseCallback[T]) PromiseReturnType[T] {

	resultchan := make(PromiseReturnType[T])

	resolvefunc := func(value T) {
		go func() {
			resultchan <- PromiseValue[T]{Value: value}
		}()
	}

	rejectfunc := func(err error) {
		go func() {
			resultchan <- PromiseValue[T]{Error: err}
		}()
	}

	go func() {
		cb(resolvefunc, rejectfunc)
	}()

	return resultchan
}

func DoRequest(url string) PromiseReturnType[*resty.Response] {
	return Promise(func(resolve ResolveFunc[*resty.Response], reject RejectFunc) {
		client := resty.New()
		res, err := client.R().
			SetHeader("Accept", "application/json").
			Get(url)

		if err != nil {
			resolve(res)
		} else {
			reject(err)
		}
	})
}
