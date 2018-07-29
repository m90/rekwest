package rekwest

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type responseType struct {
	OK     bool   `json:"ok" xml:"ok"`
	Animal string `json:"animal" xml:"animal"`
}

type badTransport int

func (b badTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("i'm just a bad transport")
}

func TestRekwest(t *testing.T) {
	tests := map[string]struct {
		handler        http.HandlerFunc
		setupFunc      func(Rekwest)
		target         []interface{}
		expectedTarget []interface{}
		expectedError  error
	}{
		"default": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"ok":true, "animal":"platypus"}`))
			},
			func(r Rekwest) {},
			[]interface{}{&responseType{}},
			[]interface{}{&responseType{
				OK:     true,
				Animal: "platypus",
			}},
			nil,
		},
		"xml payload": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/xml")
				w.Write([]byte(`<responseType><ok>true</ok><animal>platypus</animal></responseType>`))
			},
			func(r Rekwest) {},
			[]interface{}{&responseType{}},
			[]interface{}{&responseType{
				OK:     true,
				Animal: "platypus",
			}},
			nil,
		},
		"method ok": {
			func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPost:
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte("ok!"))
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
			},
			func(r Rekwest) {
				r.Method(http.MethodPost)
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'o', 'k', '!'}},
			nil,
		},
		"method not ok": {
			func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPost:
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte("ok!"))
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
			},
			func(r Rekwest) {
				r.Method(http.MethodPatch).ResponseFormat(ResponseFormatBytes)
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{}},
			errors.New("request failed with status 405: method not allowed"),
		},
		"server error": {
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "zalgo", http.StatusInternalServerError)
			},
			func(r Rekwest) {},
			[]interface{}{},
			[]interface{}{},
			errors.New("request failed with status 500: zalgo"),
		},
		"bad json payload": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"animal": "platypus", "ok}`))
			},
			func(r Rekwest) {},
			[]interface{}{&responseType{}},
			[]interface{}{&responseType{}},
			errors.New("unexpected EOF"),
		},
		"bad xml payload": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml; charset=utf-8")
				w.Write([]byte(`<animal`))
			},
			func(r Rekwest) {},
			[]interface{}{&responseType{}},
			[]interface{}{&responseType{}},
			errors.New("XML syntax error on line 1: unexpected EOF"),
		},
		"basic auth": {
			func(w http.ResponseWriter, r *http.Request) {
				if user, pass, ok := r.BasicAuth(); !ok || user != "username" || pass != "secret" {
					http.Error(w, "bad credentials", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
				r.BasicAuth("username", "secret")
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'O', 'K'}},
			nil,
		},
		"basic auth bad": {
			func(w http.ResponseWriter, r *http.Request) {
				if user, pass, ok := r.BasicAuth(); !ok || user != "username" || pass != "secret" {
					http.Error(w, "bad credentials", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
				r.BasicAuth("username", "dunno")
			},
			[]interface{}{},
			[]interface{}{},
			errors.New("request failed with status 401: bad credentials"),
		},
		"bearer token": {
			func(w http.ResponseWriter, r *http.Request) {
				if token := r.Header.Get("Authorization"); token != "Bearer secret" {
					http.Error(w, "bad Authorization header", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
				r.BearerToken("secret")
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'O', 'K'}},
			nil,
		},
		"bearer token bad": {
			func(w http.ResponseWriter, r *http.Request) {
				if token := r.Header.Get("Authorization"); token != "Bearer secret" {
					http.Error(w, "bad Authorization header", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
				r.BearerToken("dunno")
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{}},
			errors.New("request failed with status 401: bad Authorization header"),
		},
		"bytes body": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				b, _ := ioutil.ReadAll(r.Body)
				if string(b) == "yes" {
					w.Write([]byte("no"))
					return
				}
				w.Write([]byte("yes"))
			},
			func(r Rekwest) {
				r.BytesBody([]byte("yes"))
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'n', 'o'}},
			nil,
		},
		"json body": {
			func(w http.ResponseWriter, r *http.Request) {
				b, _ := ioutil.ReadAll(r.Body)
				data := responseType{}
				if err := json.Unmarshal(b, &data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(data.Animal))
			},
			func(r Rekwest) {
				r.JSONBody(responseType{
					Animal: "dog",
				})
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'d', 'o', 'g'}},
			nil,
		},
		"bad json body": {
			func(w http.ResponseWriter, r *http.Request) {
				b, _ := ioutil.ReadAll(r.Body)
				data := responseType{}
				if err := json.Unmarshal(b, &data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write([]byte(data.Animal))
			},
			func(r Rekwest) {
				r.JSONBody(func() string { return "oh hey" }).ResponseFormat(ResponseFormatBytes)
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{}},
			errors.New("json: unsupported type: func() string"),
		},
		"xml body": {
			func(w http.ResponseWriter, r *http.Request) {
				b, _ := ioutil.ReadAll(r.Body)
				data := responseType{}
				if err := xml.Unmarshal(b, &data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write([]byte(data.Animal))
			},
			func(r Rekwest) {
				r.XMLBody(responseType{
					Animal: "dog",
				}).ResponseFormat(ResponseFormatBytes)
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'d', 'o', 'g'}},
			nil,
		},
		"bad xml body": {
			func(w http.ResponseWriter, r *http.Request) {
				b, _ := ioutil.ReadAll(r.Body)
				data := responseType{}
				if err := xml.Unmarshal(b, &data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write([]byte(data.Animal))
			},
			func(r Rekwest) {
				r.XMLBody(func() string { return "oh hey" }).ResponseFormat(ResponseFormatBytes)
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{}},
			errors.New("xml: unsupported type: func() string"),
		},
		"bad response format": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("ok"))
			},
			func(r Rekwest) {
				r.ResponseFormat("zalgo")
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{}},
			errors.New("found unknown response format zalgo"),
		},
		"timeout ok": {
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Millisecond)
				w.Write([]byte("ok"))
			},
			func(r Rekwest) {
				r.Timeout(time.Second)
			},
			[]interface{}{},
			[]interface{}{},
			nil,
		},
		"timeout not ok": {
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
				w.Write([]byte("ok"))
			},
			func(r Rekwest) {
				r.Timeout(time.Microsecond)
			},
			[]interface{}{},
			[]interface{}{},
			errors.New("exceeded request timeout of 1µs"),
		},
		"context ok": {
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Millisecond)
				w.Write([]byte("ok"))
			},
			func(r Rekwest) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				r.Context(ctx)
				go func() {
					time.Sleep(time.Hour)
					cancel()
				}()
			},
			[]interface{}{},
			[]interface{}{},
			nil,
		},
		"context not ok": {
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
				w.Write([]byte("ok"))
			},
			func(r Rekwest) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
				r.Context(ctx)
				go func() {
					time.Sleep(time.Hour)
					cancel()
				}()
			},
			[]interface{}{},
			[]interface{}{},
			errors.New("context deadline exceeded"),
		},
		"header": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(r.Header.Get("X-Unit-Test")))
			},
			func(r Rekwest) {
				r.ResponseFormat(ResponseFormatBytes).Header("X-Unit-Test", "ok!")
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'o', 'k', '!'}},
			nil,
		},
		"headers": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(r.Header.Get("X-One")))
				w.Write([]byte(r.Header.Get("X-Two")))
			},
			func(r Rekwest) {
				r.ResponseFormat(ResponseFormatBytes).Headers(map[string]string{
					"X-One": "1",
					"X-Two": "2",
				})
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{'1', '2'}},
			nil,
		},
		"client": {
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
				r.Client(&http.Client{
					Transport: badTransport(0),
				})
			},
			[]interface{}{&[]byte{}},
			[]interface{}{&[]byte{}},
			errors.New("i'm just a bad transport"),
		},
		"bad target type": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
			},
			[]interface{}{&[]string{}},
			[]interface{}{&[]string{}},
			errors.New("expected byte slice elem, encountered []string when decoding into target element"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(test.handler))
			defer ts.Close()
			r := New(ts.URL)
			test.setupFunc(r)
			err := r.Do(test.target...)
			if test.expectedError != nil {
				if err == nil {
					t.Errorf("Expected %v, got nil", test.expectedError)
				} else {
					if !strings.Contains(strings.TrimSpace(err.Error()), strings.TrimSpace(test.expectedError.Error())) {
						t.Errorf("Expected error %v, got %v", test.expectedError, err)
					}
				}
			} else if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			if !reflect.DeepEqual(test.expectedTarget, test.target) {
				t.Errorf("Expected %v, got %v", test.expectedTarget, test.target)
			}
		})
	}
}
