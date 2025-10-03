package ft_supabase_utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Sentinel errors for HTTP operations.
var (
	ErrMarshalRequest = errors.New("failed to marshal request")
	ErrCreateRequest  = errors.New("failed to create HTTP request")
	ErrSendRequest    = errors.New("failed to send request")
	ErrReadResponse   = errors.New("failed to read response body")
	ErrInvalidStatus  = errors.New("invalid response status")
)

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	// Ft_SupabaseSendRequest sends an HTTP request and returns the response body.
	Ft_SupabaseSendRequest(ctx context.Context, method, url string, body any, headers map[string]string) ([]byte, error)
}

// DefaultHTTPClient implements HTTPClient with standard HTTP operations.
type DefaultHTTPClient struct{}

// NewFt_SupabaseHTTPClient creates a new default HTTP client.
// Returns an HTTPClient implementation ready for use.
func NewFt_SupabaseHTTPClient() HTTPClient {
	// create and return default HTTP client
	return &DefaultHTTPClient{}
}

// PrintRaw prints any object in a formatted JSON representation.
// title is the header text to display above the output.
// data is the object to print (can be bytes, struct, map, etc.).
// Prints formatted JSON if possible, otherwise prints raw string representation.
func PrintRaw(title string, data any) {
	var (
		jsonBytes  []byte
		prettyJSON bytes.Buffer
		err        error
	)

	// print header
	fmt.Printf("=== %s ===\n", title)

	// handle byte array input
	if byteData, ok := data.([]byte); ok {
		// attempt to indent raw bytes
		if err = json.Indent(&prettyJSON, byteData, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
		} else {
			fmt.Println(string(byteData))
		}
	} else {
		// marshal object to JSON
		jsonBytes, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting data: %v\n", err)
			fmt.Printf("%+v\n", data)
		} else {
			fmt.Println(string(jsonBytes))
		}
	}

	// print footer
	fmt.Println("=============================")
}

// Ft_SupabaseSendRequest sends an HTTP request to Supabase API and returns the response body.
// ctx is the context for request cancellation and timeout.
// method is the HTTP method (GET, POST, PUT, DELETE, etc.).
// url is the full URL endpoint.
// body is the request body to send (can be nil for GET requests).
// headers is a map of HTTP headers to set.
// Returns the response body as bytes or an error.
func (c *DefaultHTTPClient) Ft_SupabaseSendRequest(ctx context.Context, method, url string, body any, headers map[string]string) ([]byte, error) {
	var (
		jsonData  []byte
		req       *http.Request
		client    *http.Client
		resp      *http.Response
		bodyBytes []byte
		err       error
	)

	// marshal body to JSON if provided
	if body != nil {
		jsonData, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMarshalRequest, err)
		}
	}

	// create HTTP request with context
	req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateRequest, err)
	}

	// set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// initialize HTTP client
	client = &http.Client{}

	// send request
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSendRequest, err)
	}
	defer resp.Body.Close()

	// read response body
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadResponse, err)
	}

	// print raw response for debugging
	PrintRaw(fmt.Sprintf("RAW %s RESPONSE", method), bodyBytes)

	// check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%w (status %d): %s", ErrInvalidStatus, resp.StatusCode, string(bodyBytes))
	}

	// return response body
	return bodyBytes, nil
}
