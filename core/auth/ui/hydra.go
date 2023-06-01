package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HydraClient struct {
	httpClient *http.Client
	host       string
}

func NewHydraClient(host string) *HydraClient {
	return &HydraClient{
		// TODO(giolekva): trust selfsigned-root-ca automatically on pods
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		host,
	}
}

type loginResp struct {
	RedirectTo       string `json:"redirect_to"`
	Error            string `json:"error"`
	ErrorDebug       string `json:"error_debug"`
	ErrorDescription string `json:"error_description"`
	StatusCode       int    `json:"status_code"`
}

func (c *HydraClient) LoginAcceptChallenge(challenge, subject string) (string, error) {
	req := &http.Request{
		Method: http.MethodPut,
		URL: &url.URL{
			Scheme:   "http",
			Host:     c.host,
			Path:     "/admin/oauth2/auth/requests/login/accept",
			RawQuery: fmt.Sprintf("login_challenge=%s", challenge),
		},
		Header: map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		// TODO(giolekva): user stable userid instead
		Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`
{
    "subject": "%s",
    "remember": true,
    "remember_for": 3600
}`, subject))),
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	var r loginResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if r.Error != "" {
		return "", errors.New(r.Error)
	}
	return r.RedirectTo, nil
}

func (c *HydraClient) LoginRejectChallenge(challenge, message string) (string, error) {
	req := &http.Request{
		Method: http.MethodPut,
		URL: &url.URL{
			Scheme:   "http",
			Host:     c.host,
			Path:     "/admin/oauth2/auth/requests/login/reject",
			RawQuery: fmt.Sprintf("login_challenge=%s", challenge),
		},
		Header: map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`
{
    "error": "login_required %s"
}`, message))),
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	var r loginResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if r.Error != "" {
		return "", errors.New(r.Error)
	}
	return r.RedirectTo, nil
}

type RequestedConsent struct {
	Challenge       string   `json:"challenge"`
	Subject         string   `json:"subject"`
	RequestedScopes []string `json:"requested_scope"`
}

func (c *HydraClient) GetConsentChallenge(challenge string) (RequestedConsent, error) {
	var consent RequestedConsent
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s/admin/oauth2/auth/requests/consent?consent_challenge=%s", c.host, challenge))
	if err != nil {
		return consent, err
	}
	err = json.NewDecoder(resp.Body).Decode(&consent)
	return consent, err
}

type consentAcceptReq struct {
	GrantScope []string `json:"grant_scope"`
	Session    session  `json:"session"`
}

type session struct {
	IDToken map[string]string `json:"id_token"`
}

type consentAcceptResp struct {
	RedirectTo string `json:"redirect_to"`
}

func (c *HydraClient) ConsentAccept(challenge string, scopes []string, idToken map[string]string) (string, error) {
	accept := consentAcceptReq{scopes, session{idToken}}
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(accept); err != nil {
		return "", err
	}
	req := &http.Request{
		Method: http.MethodPut,
		URL: &url.URL{
			Scheme:   "http",
			Host:     c.host,
			Path:     "/admin/oauth2/auth/requests/consent/accept",
			RawQuery: fmt.Sprintf("challenge=%s", challenge),
		},
		Header: map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(&data),
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	var r consentAcceptResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.RedirectTo, err
}
