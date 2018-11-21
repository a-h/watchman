package dataprotection

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

// ErrRequestBodyMustBeJSON is returned when the request body must be JSON and it isn't.
var ErrRequestBodyMustBeJSON = errors.New("if a request body is present, it must be JSON")

// RequestBodyMustBeJSON is a request filter which returns an error if the request body (if present) is not JSON.
func RequestBodyMustBeJSON(r *http.Request) error {
	if r.Body != nil {
		var m map[string]interface{}
		d := json.NewDecoder(r.Body)
		if d.Decode(&m) != nil {
			return ErrRequestBodyMustBeJSON
		}
	}
	return nil
}

// ErrResponseBodyMustBeJSON is returned when the response body must be JSON, and it isn't.
var ErrResponseBodyMustBeJSON = errors.New("if a response body is present, it must be JSON")

// ResponseBodyMustBeJSON is a response filter which returns an error if the response body (if present) is not JSON.
func ResponseBodyMustBeJSON(r *http.Request, resp *http.Response) error {
	if resp != nil && resp.Body != nil {
		var m map[string]interface{}
		d := json.NewDecoder(resp.Body)
		if d.Decode(&m) != nil {
			return ErrResponseBodyMustBeJSON
		}
	}
	return nil
}

// ErrRequestAuthorizationHeaderMustContainAJWT is returned when a request's authorization header must contain a JWT, and it doesn't.
var ErrRequestAuthorizationHeaderMustContainAJWT = errors.New("the request must have an authorization header containing a JWT")

// RequestAuthorizationHeaderMustContainAJWT is a request filter which returns an error if the request's Authorization header doesn't contain a JWT.
// The JWT itself is not validated against the signature.
func RequestAuthorizationHeaderMustContainAJWT(r *http.Request) error {
	var parsedJWT bool
	kf := func(t *jwt.Token) (interface{}, error) {
		parsedJWT = true
		return nil, nil
	}
	jwt.Parse(r.Header.Get("Authorization"), kf)
	if parsedJWT {
		return nil
	}
	return ErrRequestAuthorizationHeaderMustContainAJWT
}

// ErrResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT is an error returned when the JSON response should only contain the user ID from the
// Authorization token, but it contains a different user ID.
var ErrResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT = errors.New("response JSON must only contain the user ID from the authorization JWT")

// ResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT is a response filter which checks that if the response body is JSON, the JSON only contains
// the users's userId (as defined by the 'sub' claim) in the Authorization header.
func ResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT(r *http.Request, resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	var parsedJWT bool
	kf := func(t *jwt.Token) (interface{}, error) {
		parsedJWT = true
		return nil, nil
	}
	var c jwt.MapClaims
	jwt.ParseWithClaims(r.Header.Get("Authorization"), &c, kf)
	if !parsedJWT {
		return ErrRequestAuthorizationHeaderMustContainAJWT
	}
	var m map[string]interface{}
	d := json.NewDecoder(resp.Body)
	if d.Decode(&m) != nil {
		return nil
	}
	rjv := RestrictJSONValue{
		Key:   "userId",
		Value: c["sub"],
	}
	if rjv.HasKeyWithUnexpectedValue(m) {
		return ErrResponseJSONMustOnlyContainTheUserIDFromTheAuthorizationJWT
	}
	return nil
}
