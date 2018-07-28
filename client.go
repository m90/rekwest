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
	"strings"
	"time"
)

type request struct {
	client *http.Client

	errors []error

	url            string
	method         string
	body           io.Reader
	target         interface{}
	header         http.Header
	basicAuth      *credentials
	bearerToken    string
	context        context.Context
	responseFormat ResponseFormat
	timeout        *time.Duration
}

func (r *request) Errors() []error {
	return r.errors
}

func (r *request) OK() bool {
	return len(r.errors) == 0
}

func (r *request) Method(m string) Rekwest {
	r.method = m
	return r
}

func (r *request) StringBody(data string) Rekwest {
	r.body = strings.NewReader(data)
	return r
}

func (r *request) MarshalBody(data interface{}, marshalFunc func(interface{}) ([]byte, error)) Rekwest {
	b, err := marshalFunc(data)
	if err != nil {
		r.errors = append(r.errors, err)
	} else {
		r.body = bytes.NewReader(b)
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

func (r *request) Target(t interface{}) Rekwest {
	r.target = t
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

func (r *request) Do() error {
	if !r.OK() {
		return Error{r.Errors()}
	}
	timeout := context.Background()
	if r.timeout != nil {
		ctx, cancel := context.WithTimeout(context.Background(), *r.timeout)
		timeout = ctx
		defer cancel()
	}

	receive := make(chan doResult)
	defer close(receive)

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
		return timeout.Err()
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
		if r.target != nil {
			switch r.responseFormat {
			case ResponseFormatJSON:
				if err := json.NewDecoder(result.res.Body).Decode(r.target); err != nil {
					return err
				}
			case ResponseFormatXML:
				if err := xml.NewDecoder(result.res.Body).Decode(r.target); err != nil {
					return err
				}
			default:
				return fmt.Errorf("found unknown response format %s", r.responseFormat)
			}
		}
	}

	return nil
}
