package rekwest

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type responseType struct {
	OK     bool   `json:"ok"`
	Animal string `json:"animal"`
}

func TestRekwest(t *testing.T) {
	tests := map[string]struct {
		handler        http.HandlerFunc
		setupFunc      func(Rekwest)
		target         interface{}
		expectedTarget interface{}
		expectedError  error
	}{
		"default": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"ok":true, "animal":"platypus"}`))
			},
			func(r Rekwest) {},
			&responseType{},
			&responseType{
				OK:     true,
				Animal: "platypus",
			},
			nil,
		},
		"server error": {
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "zalgo", http.StatusInternalServerError)
			},
			func(r Rekwest) {},
			&responseType{},
			&responseType{},
			errors.New("request failed with status 500: zalgo"),
		},
		"basic auth": {
			func(w http.ResponseWriter, r *http.Request) {
				if user, pass, ok := r.BasicAuth(); !ok || user != "username" || pass != "secret" {
					http.Error(w, "bad credentials", http.StatusUnauthorized)
					return
				}
				w.Write([]byte("OK"))
			},
			func(r Rekwest) {
				r.BasicAuth("username", "secret")
			},
			nil,
			nil,
			nil,
		},
		"bytes body": {
			func(w http.ResponseWriter, r *http.Request) {
				b, _ := ioutil.ReadAll(r.Body)
				if string(b) == "yes" {
					w.Write([]byte("no"))
					return
				}
				w.Write([]byte("yes"))
			},
			func(r Rekwest) {
				r.BytesBody([]byte("yes")).ResponseFormat(ResponseFormatBytes)
			},
			&[]byte{},
			&[]byte{'n', 'o'},
			nil,
		},
		"bad response format": {
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("ok"))
			},
			func(r Rekwest) {
				r.ResponseFormat("zalgo")
			},
			&responseType{},
			&responseType{},
			errors.New("found unknown response format zalgo"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(test.handler))
			defer ts.Close()
			r := New(ts.URL)
			test.setupFunc(r)
			err := r.Do(test.target)
			if test.expectedError != nil {
				if err == nil {
					t.Errorf("Expected %v, got nil", test.expectedError)
				} else {
					if strings.TrimSpace(test.expectedError.Error()) != strings.TrimSpace(err.Error()) {
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
