package tg_api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Send a GET request to the endpoint
func Exchange(endpoint string) (*[]byte, error) {
	if res, e := http.Get(endpoint); e == nil {
		if body, err_b := io.ReadAll(res.Body); err_b == nil {
			return &body, nil
		} else {
			return nil, err_b
		}
	} else {
		return nil, e
	}
} // <-- Exchange(endpoint)

// Send a POST request to the endpoint with JSON data from `params`
func ExchangeWith[P any](endpoint string, params P) (*[]byte, error) {
	if body, js_e := json.Marshal(params); js_e == nil {
		// log.Println("Sending", string(body))
		rqb := bytes.NewBuffer(body)
		if res, e := http.Post(endpoint, "application/json", rqb); e == nil {
			if resp_body, err_b := io.ReadAll(res.Body); err_b == nil {
				return &resp_body, nil
			} else {
				return nil, err_b
			}
		} else {
			return nil, e
		}
	} else {
		return nil, js_e
	}
} // <-- ExchangeWith[P](endpoint, params)

// Same as `exchange`, but stores deserializes the response into T
func ExchangeInto[T any](endpoint string) (*ExchangeResult[T], error) {
	if res, err := Exchange(endpoint); err == nil {
		var ret ExchangeResult[T]
		// log.Println("Deserializing ", string(*res))
		if js_err := json.Unmarshal(*res, &ret); js_err != nil {
			return nil, js_err
		}
		return &ret, nil
	} else {
		return nil, err
	}
} // <-- ExchangeInto[T](endpoint)

// Same as `exchange_with`, but stores deserializes the response into T
func ExchangeIntoWith[T any, P any](
	endpoint string, params P,
) (*ExchangeResult[T], error) {
	if res, err := ExchangeWith(endpoint, params); err == nil {
		var ret ExchangeResult[T]
		// log.Println("Deserializing ", string(*res))
		if js_err := json.Unmarshal(*res, &ret); js_err != nil {
			return nil, js_err
		}
		return &ret, nil
	} else {
		return nil, err
	}
} // <-- ExchangeIntoWith[T, P](endpoint, params)
