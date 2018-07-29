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

Perform a POST request, encoding the JSON response into `data`:

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

### License
MIT © [Frederik Ring](http://www.frederikring.com)
