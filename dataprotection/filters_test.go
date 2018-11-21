package dataprotection

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestRequestBodyMustBeJSON(t *testing.T) {
	tests := []struct {
		name     string
		r        *http.Request
		expected error
	}{
		{
			name:     "if the request body is nil, there is no error",
			r:        &http.Request{},
			expected: nil,
		},
		{
			name: "if the request body is not JSON, an error is returned",
			r: &http.Request{
				Body: ioutil.NopCloser(bytes.NewBufferString("dfhjsdfhd  ")),
			},
			expected: ErrRequestBodyMustBeJSON,
		},
	}
	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := RequestBodyMustBeJSON(test.r)
			if actual != test.expected {
				t.Errorf("expected '%v', got '%v'", test.expected, actual)
			}
		})
	}
}

func TestResponseBodyMustBeJSON(t *testing.T) {
	tests := []struct {
		name     string
		r        *http.Request
		resp     *http.Response
		expected error
	}{
		{
			name: "no response, no problem",
		},
		{
			name: "no response body, no problem",
			resp: &http.Response{
				Body: nil,
			},
		},
		{
			name: "JSON body responses don't produce an error",
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("{}")),
			},
			expected: nil,
		},
		{
			name: "if the response body is not JSON, an error is returned",
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("sdfjhudsgfhj ")),
			},
			expected: ErrResponseBodyMustBeJSON,
		},
	}
	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := ResponseBodyMustBeJSON(test.r, test.resp)
			if actual != test.expected {
				t.Errorf("expected '%v', got '%v'", test.expected, actual)
			}
		})
	}
}

func TestRequestAuthorizationHeaderMustContainAJWT(t *testing.T) {
	header := `{ "typ": "jwt", "alg": "HS256" }`
	claims := `{ "sub": "123" }`
	signature := `sdhfjksdhfjkshfjdkshfjdk`
	basicJWT := base64.StdEncoding.EncodeToString([]byte(header)) + "." +
		base64.StdEncoding.EncodeToString([]byte(claims)) + "." +
		base64.StdEncoding.EncodeToString([]byte(signature))
	tests := []struct {
		name     string
		r        *http.Request
		expected error
	}{
		{
			name:     "if there are no headers, an error is returned",
			r:        &http.Request{},
			expected: ErrRequestAuthorizationHeaderMustContainAJWT,
		},
		{
			name: "if there is an authorization header, but it's not a JWT, an error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{"sdfd"},
				},
			},
			expected: ErrRequestAuthorizationHeaderMustContainAJWT,
		},
		{
			name: "if there is an authorization header and it contains any sort of JWT (even unsigned) no error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			expected: nil,
		},
	}
	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := RequestAuthorizationHeaderMustContainAJWT(test.r)
			if actual != test.expected {
				t.Errorf("expected '%v', got '%v'", test.expected, actual)
			}
		})
	}
}

func TestResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT(t *testing.T) {
	header := `{ "typ": "jwt", "alg": "HS256" }`
	claims := `{ "sub": "123" }`
	signature := `sdhfjksdhfjkshfjdkshfjdk`
	basicJWT := base64.StdEncoding.EncodeToString([]byte(header)) + "." +
		base64.StdEncoding.EncodeToString([]byte(claims)) + "." +
		base64.StdEncoding.EncodeToString([]byte(signature))
	tests := []struct {
		name     string
		r        *http.Request
		resp     *http.Response
		expected error
	}{
		{
			name: "no response body, no error",
			r:    &http.Request{},
			resp: &http.Response{
				Body: nil,
			},
			expected: nil,
		},
		{
			name: "response body, with no headers returns an error",
			r:    &http.Request{},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("sdfjdskhjk")),
			},
			expected: ErrRequestAuthorizationHeaderMustContainAJWT,
		},
		{
			name: "response body, with an invalid authorization header returns an error",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{"sdfd"},
				},
			},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("sdfjdskhjk")),
			},
			expected: ErrRequestAuthorizationHeaderMustContainAJWT,
		},
		{
			name: "if there is an authorization header, and the response body contains JWT without the 'userId' key, no error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			expected: nil,
		},
		{
			name: "if there is an authorization header, and the response body is not JSON, no error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString("userId, but it's not JSON")),
			},
			expected: nil,
		},
		{
			name: "if there is an authorization header, and the response body contains JWT with the expected 'userId' key value, no error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString(`{ "userId": "123" }`)),
			},
			expected: nil,
		},
		{
			name: "if there is an authorization header, and the response body contains JWT with the expected 'userId' key value in a subdocument, no error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString(`{ "subdocument": { "userId": "123" } }`)),
			},
			expected: nil,
		},
		{
			name: "if there is an authorization header, and the response body contains JWT with the 'userId' key but with an unexpected value, an error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString(`{ "userId": "not the user" }`)),
			},
			expected: ErrResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT,
		},
		{
			name: "if there is an authorization header, and the response body contains JWT with the 'userId' key but with an unexpected value in the subdocument, an error is returned",
			r: &http.Request{
				Header: map[string][]string{
					"Authorization": []string{basicJWT},
				},
			},
			resp: &http.Response{
				Body: ioutil.NopCloser(bytes.NewBufferString(`{ "subdoc": { "userId": "not the user" } }`)),
			},
			expected: ErrResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT,
		},
	}
	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := ResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT(test.r, test.resp)
			if actual != test.expected {
				t.Errorf("expected '%v', got '%v'", test.expected, actual)
			}
		})
	}
}
