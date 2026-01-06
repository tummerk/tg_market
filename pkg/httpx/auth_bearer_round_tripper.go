package httpx

import (
	"context"
	"fmt"
	"net/http"
)

type authenticator interface {
	Authenticate(context.Context) error
	BearerToken() string
}

type AuthBearerRoundTripper struct {
	next          http.RoundTripper
	authenticator authenticator
}

func NewAuthBearerRoundTripper(
	next http.RoundTripper,
	authenticator authenticator,
) AuthBearerRoundTripper {
	return AuthBearerRoundTripper{
		next:          next,
		authenticator: authenticator,
	}
}

func (rt AuthBearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.authenticator.BearerToken() == "" {
		if err := rt.authenticator.Authenticate(req.Context()); err != nil {
			return nil, fmt.Errorf("authenticator.Authenticate: %w", err)
		}
	}

	rt.setAuthorizationHeader(req)

	resp, err := rt.next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("next.RoundTrip: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		if err = rt.authenticator.Authenticate(req.Context()); err != nil {
			return nil, fmt.Errorf("authenticator.Authenticate: %w", err)
		}

		rt.setAuthorizationHeader(req)

		return rt.next.RoundTrip(req) //nolint:wrapcheck
	}

	return resp, nil
}

func (rt AuthBearerRoundTripper) setAuthorizationHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+rt.authenticator.BearerToken())
}
