package debugapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTTPStatusErrorCode(t *testing.T) {
	if ok := IsHTTPStatusErrorCode(NewHTTPStatusError(http.StatusBadGateway), http.StatusBadGateway); !ok {
		t.Fatal("got false")
	}
	if ok := IsHTTPStatusErrorCode(NewHTTPStatusError(http.StatusBadGateway), http.StatusInternalServerError); ok {
		t.Fatal("got true")
	}
	if ok := IsHTTPStatusErrorCode(nil, http.StatusTeapot); ok {
		t.Fatal("got true")
	}
	if ok := IsHTTPStatusErrorCode(io.EOF, http.StatusTeapot); ok {
		t.Fatal("got true")
	}
}

func TestResponseErrorHandler(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.Handler
		err     error
	}{
		{
			name:    "blank",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}),
		},
		{
			name: "status ok",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
		{
			name: "status created",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}),
		},
		{
			name: "status only",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			}),
			err: NewHTTPStatusError(http.StatusBadRequest),
		},
		{
			name: "status only 2",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
			err: NewHTTPStatusError(http.StatusInternalServerError),
		},
		{
			name: "no data",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", contentType)
			}),
			err: NewHTTPStatusError(http.StatusInternalServerError),
		},
		{
			name: "no message",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", contentType)
				_, _ = w.Write(encodeMessageResponse(t, ""))
			}),
			err: NewHTTPStatusError(http.StatusInternalServerError),
		},
		{
			name: "custom message",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write(encodeMessageResponse(t, "custom message"))
			}),
			err: fmt.Errorf("response message %q: status: %w", "custom message", NewHTTPStatusError(http.StatusInternalServerError)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.handler.ServeHTTP(recorder, nil)

			gotErr := responseErrorHandler(recorder.Result())

			if tc.err == nil && gotErr == nil {
				return // all fine
			}

			var e *HTTPStatusError
			if !errors.As(gotErr, &e) {
				t.Fatalf("got error %v, want %v", gotErr, tc.err)
			} else if e.Code != recorder.Code {
				t.Fatalf("got error code %v, want %v", e.Code, recorder.Code)
			}

			gotErrMessage := gotErr.Error()
			wantErrMessage := tc.err.Error()
			if gotErrMessage != wantErrMessage {
				t.Fatalf("got error message %q, want %q", gotErrMessage, wantErrMessage)
			}
		})
	}
}

func encodeMessageResponse(t *testing.T, message string) []byte {
	t.Helper()

	data, err := json.Marshal(messageResponse{
		Message: message,
	})
	if err != nil {
		t.Fatal(err)
	}

	return data
}
