# rekwest
[![Build Status](https://travis-ci.org/m90/rekwest.svg?branch=master)](https://travis-ci.org/m90/rekwest)
[![godoc](https://godoc.org/github.com/m90/rekwest?status.svg)](http://godoc.org/github.com/m90/rekwest)

> Fluent HTTP request client

## Installation

Use `go get`:

```sh
$ go get github.com/m90/rekwest
```

## Example

Perform a POST request, encoding the JSON response into `data`, by calling `Do(targets ...interface{})` on the previously constructed request:

```go
req := rekwest.New("https://www.example.com/api/create-animal").
    Method(http.MethodPost).
    JSONBody(map[string]interface{}{
        "kind":     "platypus",
        "flappers": true,
    }).
    BasicAuth("username", "secret")

data := responseType{}
if err := req.Do(&data); err != nil {
    panic(err)
}
```

### Features

#### Authentication

Use `BasicAuth(username, password string)` or `BearerToken(token string)` to send `Authorization` headers:

```go
rekwest.New("https://www.example.com/api").BasicAuth("user", "pass")

rekwest.New("https://www.example.com/api").BearerToken("my-token")
```

### Context

Add a `context.Context` using `Context(ctx context.Context)`:

```go
timeout, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

rekwest.New("https://www.example.com/api").Context(timeout).Do()
```

### Timeout

Set a duration for when the request is supposed to time out using `Timeout(value time.Duration)`:

```go
rekwest.New("https://www.example.com/api").Timeout(time.Second)
```

### Headers

Set header values using `Header(key, value string)` or `Headers(headers map[string]string)`:

```go
r := rekwest.New("https://www.example.com/api").Header("X-Foo", "bar")
r.Headers(map[string]string{
	"X-Bar":   "foo",
	"X-Other": "another one",
})
```

### HTTP Client

Use a custom `http.Client` instance by passing it to `Client(client *http.Client)`:

```go
rekwest.New("https://www.example.com/api").Client(&http.Client{
	Timeout: time.Second,
})
```

### Response content type

Use `ResponseFormat(format ResponseFormat)` in case you want to specify the expected payload:

```go
json := rekwest.New("https://www.example.com/api").ResponseFormat(rekwest.ResponseFormatJSON)
data := responseType{}
err := json.Do(&data)
```

Available formats are `ResponseFormatJSON`, `ResponseFormatXML` and `ResponseFormatBytes`. If no value is set, `rekwest` will try to read the responses `Content-Type` header and act accordingly. If none is sent, the response body will be treated as type `[]byte`.

For JSON and XML, the correct `Accept` header will be automatically set.

### Request body Marshaling

Request payloads can automatically be marshalled into the desired format using `JSONBody(data interface{})`, `XMLBody(data interface{})` and `MarshalBody(data interface{}, marshalFunc func(interface{}) ([]byte, error))`:

```go
rekwest.New("https://www.example.com/api/create-animal").
    Method(http.MethodPost).
    JSONBody(map[string]interface{}{
        "kind":     "platypus",
        "flappers": true,
    })
```

Alternatively an `io.Reader` can be passed to `Body(data io.Reader)`.

### License
MIT © [Frederik Ring](http://www.frederikring.com)
