package rekwest

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type responseType struct {
	OK     bool   `json:"ok"`
	Animal string `json:"animal"`
}

func TestRekwest(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user, pass, ok := r.BasicAuth(); !ok || user != "username" || pass != "secret" {
				http.Error(w, "bad credentials", http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"ok":true, "animal":"platypus"}`))
		}))

		data := responseType{}
		expected := responseType{
			OK:     true,
			Animal: "platypus",
		}
		err := New(ts.URL).
			Target(&data).
			BasicAuth("username", "secret").
			Do()
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if !reflect.DeepEqual(expected, data) {
			t.Errorf("Expected %v, got %v", expected, data)
		}
	})
}
