package dataprotection

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"
)

const pkg = "watchman"

// A RequestFilter filters requests prior to being forwarded.
type RequestFilter func(r *http.Request) error

// A ResponseFilter filters responses prior to returning a value.
type ResponseFilter func(r *http.Request, resp *http.Response) error

// ProtectionMode determines whether the filter warns (default) or blocks.
type ProtectionMode int

const (
	// ProtectionModeWarn only warns, and doesn't affect the operation of the child handler.
	ProtectionModeWarn ProtectionMode = iota
	// ProtectionModeBlock prevents the request from being accepted, or response from being returned, instead
	// returning a 401 unauthorized error.
	ProtectionModeBlock
)

// New creates a new handler which watches inbound and outbound HTTP requests to determine
// whether they contain unexpected data.
func New(next http.Handler, requestFilters []RequestFilter,
	responseFilters []ResponseFilter, mode ProtectionMode) Middleware {
	return Middleware{
		Next:            next,
		RequestFilters:  requestFilters,
		ResponseFilters: responseFilters,
		Mode:            mode,
		Error:           BasicError,
		Now:             time.Now,
		Logger:          func(s string) { os.Stderr.WriteString(s) },
	}
}

// Middleware which examines the HTTP requests.
type Middleware struct {
	Next            http.Handler
	RequestFilters  []RequestFilter
	ResponseFilters []ResponseFilter
	Mode            ProtectionMode
	Error           ErrorFunc
	Now             func() time.Time
	Logger          func(line string)
}

// ErrorFunc is the content returned if the request or response is blocked.
type ErrorFunc func(w http.ResponseWriter, r *http.Request)

// BasicError returns a basic HTTP error in the case that the request is blocked.
func BasicError(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "blocked", http.StatusForbidden)
}

type logEntry struct {
	Time       time.Time `json:"time"`
	Package    string    `json:"pkg"`
	Method     string    `json:"method"`
	UserAgent  string    `json:"userAgent"`
	RemoteAddr string    `json:"remoteAddr"`
	URL        string    `json:"url"`
	Error      string    `json:"err"`
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var errs []error
	for _, reqFilter := range m.RequestFilters {
		if err := reqFilter(r); err != nil {
			errs = append(errs, err)
		}
	}
	err := joinErrs(errs)
	if err != nil {
		m.Log(r, err)
		if m.Mode == ProtectionModeBlock {
			m.Error(w, r)
			return
		}
	}
	errs = nil
	err = nil

	// Execute the child handler.
	rec := httptest.NewRecorder()
	m.Next.ServeHTTP(rec, r)

	// Validate the output.
	for _, respFilter := range m.ResponseFilters {
		if err := respFilter(r, rec.Result()); err != nil {
			errs = append(errs, err)
		}
	}
	err = joinErrs(errs)
	if err != nil {
		m.Log(r, err)
		if m.Mode == ProtectionModeBlock {
			m.Error(w, r)
			return
		}
	}

	// Write the output.
	for k, v := range rec.Header() {
		for _, vv := range v {
			w.Header().Set(k, vv)
		}
	}
	w.WriteHeader(rec.Code)
	w.Write(rec.Body.Bytes())
	return
}

func joinErrs(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	var joined []string
	for _, err := range errs {
		if err != nil {
			joined = append(joined, err.Error())
		}
	}
	return errors.New(strings.Join(joined, ", "))
}

// Log the request to the configured logger.
func (m Middleware) Log(r *http.Request, err error) {
	if r == nil {
		r = &http.Request{}
	}
	entry := logEntry{
		Time:       m.Now(),
		Package:    pkg,
		Method:     r.Method,
		UserAgent:  r.UserAgent(),
		RemoteAddr: firstNonEmpty(r.Header.Get("X-Forwarded-For"), r.RemoteAddr),
	}
	if r.URL != nil {
		entry.URL = r.URL.String()
	}
	if err != nil {
		entry.Error = err.Error()
	}
	bytes, err := json.Marshal(entry)
	if err != nil {
		m.Logger(`{ "pkg": "watchman", "err": "json logger error" }` + "\n")
		return
	}
	m.Logger(string(bytes) + "\n")
}
