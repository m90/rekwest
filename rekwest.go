package rekwest

import (
	"context"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
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
		responseFormat: ResponseFormatContentType,
	}
}

// Rekwest is a chainable interface for building and performing HTTP requests.
type Rekwest interface {
	// Method sets the request Method.
	Method(string) Rekwest
	// Body sets the request body.
	Body(io.Reader) Rekwest
	// bytesBody uses the given byte array as the request body.
	BytesBody([]byte) Rekwest
	// MarshalBody uses the given marshal func to marshal the given data into the
	// request body. For JSON and XML payloads, you can use the JSONBody and
	// XMLBody methods.
	MarshalBody(interface{}, func(interface{}) ([]byte, error)) Rekwest
	// JSONBody marshals the given data into JSON and uses it as the request body.
	JSONBody(interface{}) Rekwest
	// XMLBody marshals the given data into XML and uses it as the request body.
	XMLBody(interface{}) Rekwest
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
	// Errors returns all errors that occurred when building the request.
	Errors() []error
	// OK returns true if no errors have been encountered when building the request.
	OK() bool
	// Do performs the request and returns possible errors.
	// The response body will encoded onto the passed target if given.
	Do(...interface{}) error
}

// ResponseFormat is a string describing the expected encoding
// of the response.
type ResponseFormat string

// A list of supported `ResponseFormat`s.
const (
	ResponseFormatContentType ResponseFormat = "content-type"
	ResponseFormatJSON        ResponseFormat = "json"
	ResponseFormatXML         ResponseFormat = "xml"
	ResponseFormatBytes       ResponseFormat = "bytes"
)

type targetFormat string

const (
	targetFormatJSON  targetFormat = "json"
	targetFormatXML   targetFormat = "xml"
	targetFormatBytes targetFormat = "bytes"
)

func inferTargetFormat(contentType string) (targetFormat, error) {
	m, _, err := mime.ParseMediaType(contentType)
	switch m {
	case "application/json":
		return targetFormatJSON, err
	case "text/xml", "application/xml":
		return targetFormatXML, err
	default:
		return targetFormatBytes, err
	}
}

const (
	acceptJSON      = "application/json"
	acceptXML       = "text/xml, application/xml"
	contentTypeJSON = "application/json"
	contentTypeXML  = "application/xml"
)

// MultiError is a basic wrapper around multiple errors.
type MultiError struct {
	Errors []error
}

func (e MultiError) Error() string {
	var collected []string
	for _, err := range e.Errors {
		collected = append(collected, err.Error())
	}
	return strings.Join(collected, ", ")
}

func (e *MultiError) append(errors ...error) {
	for _, err := range errors {
		e.Errors = append(e.Errors, err)
	}
}
