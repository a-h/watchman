package dataprotection

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestMiddleware(t *testing.T) {
	errRequestFails := errors.New("all requests fail")
	requestFails := func(r *http.Request) error {
		return errRequestFails
	}
	fixedTime := func() time.Time { return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC) }

	tests := []struct {
		name               string
		r                  *http.Request
		requestFilters     []RequestFilter
		responseFilters    []ResponseFilter
		mode               ProtectionMode
		handler            http.HandlerFunc
		expected           *http.Response
		expectedLogEntries []string
	}{
		{
			name: "no particular request, no problem",
			r:    &http.Request{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("OK"))
			},
			expected: &http.Response{
				StatusCode: 200,
				Header: map[string][]string{
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				Body: ioutil.NopCloser(bytes.NewBufferString("OK")),
			},
		},
		{
			name: "request filters can block the request",
			r:    &http.Request{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("OK"))
			},
			requestFilters: []RequestFilter{
				requestFails,
			},
			mode:     ProtectionModeBlock,
			expected: blockedResponse(),
			expectedLogEntries: []string{
				`{"time":"2000-01-01T00:00:00Z","pkg":"watchman","method":"","userAgent":"","remoteAddr":"","url":"","err":"all requests fail"}` + "\n",
			},
		},
		{
			name: "request filters can just log the request",
			r:    &http.Request{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("OK"))
			},
			requestFilters: []RequestFilter{
				requestFails,
			},
			mode: ProtectionModeWarn,
			expected: &http.Response{
				StatusCode: 200,
				Header: map[string][]string{
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				Body: ioutil.NopCloser(bytes.NewBufferString("OK")),
			},
			expectedLogEntries: []string{
				`{"time":"2000-01-01T00:00:00Z","pkg":"watchman","method":"","userAgent":"","remoteAddr":"","url":"","err":"all requests fail"}` + "\n",
			},
		},
		{
			name: "response filters can block the response",
			r:    &http.Request{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("OK"))
			},
			responseFilters: []ResponseFilter{
				ResponseBodyMustBeJSON,
			},
			mode:     ProtectionModeBlock,
			expected: blockedResponse(),
			expectedLogEntries: []string{
				`{"time":"2000-01-01T00:00:00Z","pkg":"watchman","method":"","userAgent":"","remoteAddr":"","url":"","err":"if a response body is present, it must be JSON"}` + "\n",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := &requestRecorder{
				next: tc.handler,
			}
			mw := New(rr, tc.requestFilters, tc.responseFilters, tc.mode)
			mw.Now = fixedTime
			var logLines []string
			mw.Logger = func(line string) {
				logLines = append(logLines, line)
			}
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, tc.r)
			if err := areEqual(tc.expected, w.Result()); err != nil {
				t.Error(err)
				t.Error("Expected")
				t.Error(tc.expected)
				t.Error("Got")
				t.Error(w.Result())
			}
			if !reflect.DeepEqual(tc.expectedLogEntries, logLines) {
				t.Errorf("expected log entries:\n'%v'\ngot:\n'%v'", tc.expectedLogEntries, logLines)
			}
		})
	}
}

func blockedResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusForbidden,
		Header: map[string][]string{
			"X-Content-Type-Options": []string{"nosniff"},
			"Content-Type":           []string{"text/plain; charset=utf-8"},
		},
		Body: ioutil.NopCloser(bytes.NewBufferString("blocked\n")),
	}
}

func areEqual(r1, r2 *http.Response) error {
	if r1.StatusCode != r2.StatusCode {
		return fmt.Errorf("expected status code %d to match %d", r1.StatusCode, r2.StatusCode)
	}
	if len(r1.Header) != len(r2.Header) {
		return fmt.Errorf("expected header length %d to match %d", len(r1.Header), len(r2.Header))
	}
	r1Body, err1 := ioutil.ReadAll(r1.Body)
	r2Body, err2 := ioutil.ReadAll(r2.Body)
	if err1 != err2 {
		return fmt.Errorf("expected read body error '%v' to match '%v'", err1, err2)
	}
	if !reflect.DeepEqual(r1Body, r2Body) {
		return fmt.Errorf("expected body error '%s' to match '%s'", r1Body, r2Body)
	}
	return nil
}

type requestRecorder struct {
	R    *http.Request
	next http.HandlerFunc
}

func (rr *requestRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rr.R = r
	rr.next(w, r)
}
