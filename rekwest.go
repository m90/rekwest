package rekwest

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Rekwest is a chainable interface for building and performing HTTP requests.
type Rekwest interface {
	// Method sets the request Method.
	Method(string) Rekwest
	// Body sets the request body.
	Body(io.Reader) Rekwest
	// Target is used to decode the response into. It should be a pointer type
	// or the changes will not be reflected.
	Target(interface{}) Rekwest
	// Header sets the request header of the given key to the given value.
	Header(string, string) Rekwest
	// Headers sets the request headers for all key/value pairs in the
	// given map.
	Headers(map[string]string) Rekwest
	// BasicAuth ensures the given basic auth credentials will be used
	// when performing the request.
	BasicAuth(string, string) Rekwest
	// BearerToken ensures Authorization headers with the given bearer token
	// will be sent.
	BearerToken(string) Rekwest
	// Context adds a context to the request. In case the context hits the
	// cancellation deadline before the request can be performed, `Do` will return
	// the context's error.
	Context(context.Context) Rekwest
	// ResponseFormat sets the expected response format. It can be set to
	// ResponseFormatJSON or ResponseFormatXML.
	ResponseFormat(ResponseFormat) Rekwest
	// Timeout sets a timeout value for performing the request. The countdown
	// starts when calling `Do`.
	Timeout(time.Duration) Rekwest
	// Client ensures the given *http.Client will be used for performing the
	// request when calling `Do`.
	Client(*http.Client) Rekwest
	// Do performs the request and returns possible errors.
	Do() error
}

// ResponseFormat is a string describing the expected encoding
// of the response.
type ResponseFormat string

// A list of supported `ResponseFormat`s.
const (
	ResponseFormatJSON ResponseFormat = "json"
	ResponseFormatXML  ResponseFormat = "xml"
)

// New creates a new Rekwest that will perform requests against the given URL.
// It defaults to performing GET requests and no body, expecting JSON to be sent
// in return.
func New(url string) Rekwest {
	return &request{
		client:         http.DefaultClient,
		url:            url,
		method:         http.MethodGet,
		header:         http.Header{},
		context:        context.Background(),
		responseFormat: ResponseFormatJSON,
	}
}
