package peercount

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	contentType = "application/json; charset=utf-8"
)

// request handles the HTTP request response cycle. It JSON encodes the request
// body, creates an HTTP request with provided method on a path with required
// headers, sets current request rate information to the Client and decodes
// request body if the v argument is not nil and content type is application/json.
func request(method, path string, body, v interface{}) (err error) {
	var bodyBuffer io.ReadWriter
	if body != nil {
		bodyBuffer = new(bytes.Buffer)
		if err = encodeJSON(bodyBuffer, body); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, path, bodyBuffer)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer drain(resp.Body)

	if err = responseErrorHandler(resp); err != nil {
		return err
	}

	if v != nil && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return json.NewDecoder(resp.Body).Decode(&v)
	}
	return nil
}

// encodeJSON writes a JSON-encoded v object to the provided writer with SetEscapeHTML set to false.
func encodeJSON(w io.Writer, v interface{}) (err error) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// drain discards all of the remaining data from the reader and closes it, asynchronously.
func drain(r io.ReadCloser) {
	go func() {
		// Panicking here does not put data in an inconsistent state.
		defer func() {
			_ = recover()
		}()

		_, _ = io.Copy(ioutil.Discard, r)
		r.Close()
	}()
}

// responseErrorHandler returns an error based on the HTTP status code, or
// nil if the status code is from 200 to 299.
func responseErrorHandler(r *http.Response) (err error) {
	if r.StatusCode/100 == 2 {
		return nil
	}
	switch r.StatusCode {
	case http.StatusBadRequest:
		return decodeBadRequest(r)
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrTooManyRequests
	case http.StatusInternalServerError:
		return ErrInternalServerError
	case http.StatusServiceUnavailable:
		return ErrMaintenance
	default:
		return errors.New(strings.ToLower(r.Status))
	}
}

// decodeBadRequest parses the body of HTTP response that contains
// a list of errors as the result of bad request data.
func decodeBadRequest(r *http.Response) (err error) {

	type badRequestResponse struct {
		Errors []string `json:"errors"`
	}

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return NewBadRequestError("bad request")
	}
	var e badRequestResponse
	if err = json.NewDecoder(r.Body).Decode(&e); err != nil {
		if err == io.EOF {
			return NewBadRequestError("bad request")
		}
		return err
	}
	return NewBadRequestError(e.Errors...)
}
