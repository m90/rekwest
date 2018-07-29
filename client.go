package rekwest

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"
)

type request struct {
	client *http.Client

	multiErr MultiError

	url            string
	method         string
	body           io.Reader
	header         http.Header
	basicAuth      *credentials
	bearerToken    string
	context        context.Context
	responseFormat ResponseFormat
	timeout        *time.Duration
}

func (r *request) Errors() []error {
	return r.multiErr.Errors
}

func (r *request) OK() bool {
	return len(r.multiErr.Errors) == 0
}

func (r *request) Method(m string) Rekwest {
	r.method = m
	return r
}

func (r *request) BytesBody(data []byte) Rekwest {
	return r.Body(bytes.NewReader(data))
}

func (r *request) MarshalBody(data interface{}, marshalFunc func(interface{}) ([]byte, error)) Rekwest {
	b, err := marshalFunc(data)
	if err != nil {
		r.multiErr.append(err)
	} else {
		return r.BytesBody(b)
	}
	return r
}

func (r *request) JSONBody(data interface{}) Rekwest {
	return r.MarshalBody(data, json.Marshal)
}

func (r *request) XMLBody(data interface{}) Rekwest {
	return r.MarshalBody(data, xml.Marshal)
}

func (r *request) Body(b io.Reader) Rekwest {
	r.body = b
	return r
}

func (r *request) Header(key, value string) Rekwest {
	r.header.Add(key, value)
	return r
}

func (r *request) Headers(headers map[string]string) Rekwest {
	for key, value := range headers {
		r.header.Add(key, value)
	}
	return r
}

type credentials struct {
	userName, password string
}

func (r *request) BasicAuth(username, password string) Rekwest {
	r.basicAuth = &credentials{username, password}
	return r
}

func (r *request) BearerToken(token string) Rekwest {
	r.bearerToken = token
	return r
}

func (r *request) Context(ctx context.Context) Rekwest {
	r.context = ctx
	return r
}

func (r *request) ResponseFormat(format ResponseFormat) Rekwest {
	switch format {
	case ResponseFormatJSON:
		r.Header("Accept", acceptJSON)
	case ResponseFormatXML:
		r.Header("Accept", acceptXML)
	}
	r.responseFormat = format
	return r
}

func (r *request) Timeout(value time.Duration) Rekwest {
	r.timeout = &value
	return r
}

func (r *request) Client(client *http.Client) Rekwest {
	r.client = client
	return r
}

type doResult struct {
	res *http.Response
	err error
}

func (r *request) Do(targets ...interface{}) error {
	if !r.OK() {
		return r.multiErr
	}

	timeout := context.Background()
	if r.timeout != nil {
		ctx, cancel := context.WithTimeout(context.Background(), *r.timeout)
		timeout = ctx
		defer cancel()
	}

	receive := make(chan doResult)

	go func() {
		req, reqErr := http.NewRequest(r.method, r.url, r.body)
		if reqErr != nil {
			receive <- doResult{nil, reqErr}
			return
		}
		for key, value := range r.header {
			req.Header.Set(key, value[0])
		}

		if r.basicAuth != nil {
			req.SetBasicAuth(r.basicAuth.userName, r.basicAuth.password)
		}

		if r.bearerToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.bearerToken))
		}
		res, err := r.client.Do(req)
		receive <- doResult{res, err}
	}()

	select {
	case <-timeout.Done():
		return fmt.Errorf("exceeded request timeout of %v", r.timeout)
	case <-r.context.Done():
		return r.context.Err()
	case result := <-receive:
		if result.err != nil {
			return result.err
		}

		if result.res.Body != nil {
			defer result.res.Body.Close()
		}

		if result.res.StatusCode >= http.StatusBadRequest {
			b, err := ioutil.ReadAll(result.res.Body)
			if err != nil {
				return fmt.Errorf("request failed with status %d: %s", result.res.StatusCode, err)
			}
			return fmt.Errorf("request failed with status %d: %s", result.res.StatusCode, string(b))
		}

		for _, target := range targets {
			var format targetFormat
			switch r.responseFormat {
			case ResponseFormatJSON, ResponseFormatXML, ResponseFormatBytes:
				format = targetFormat(r.responseFormat)
			case ResponseFormatContentType:
				f, err := inferTargetFormat(result.res.Header.Get("Content-Type"))
				if err != nil {
					r.multiErr.append(err)
				} else {
					format = f
				}
			default:
				r.multiErr.append(fmt.Errorf("found unknown response format %s", r.responseFormat))
			}

			switch format {
			case targetFormatJSON:
				if err := json.NewDecoder(result.res.Body).Decode(target); err != nil {
					r.multiErr.append(err)
				}
			case targetFormatXML:
				if err := xml.NewDecoder(result.res.Body).Decode(target); err != nil {
					r.multiErr.append(err)
				}
			case targetFormatBytes:
				b, err := ioutil.ReadAll(result.res.Body)
				if err != nil {
					r.multiErr.append(err)
				}
				v := reflect.ValueOf(target)
				if k := v.Kind(); k != reflect.Ptr {
					r.multiErr.append(fmt.Errorf("expected pointer kind, encountered %v when decoding into target element", k))
					break
				}
				if s := v.Elem().Type().String(); s != "[]uint8" {
					r.multiErr.append(fmt.Errorf("expected byte slice elem, encountered %s when decoding into target element", s))
					break
				}
				v.Elem().Set(reflect.ValueOf(b))
			}
		}
	}

	if !r.OK() {
		return r.multiErr
	}
	return nil
}
