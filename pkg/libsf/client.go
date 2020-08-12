package libsf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type (
	// A Client defines all interactions that can be performed on a StandardFile server.
	Client interface {
		// GetAuthParams returns the parameters of a user from StandardFile server.
		GetAuthParams(email string) (Auth, error)
		// Login connects the Client to the StandardFile server.
		Login(email, password string) error
		// BearerToken returns the authentication used for requests sent to the StandardFile server.
		BearerToken() string
		// SetBearerToken sets the authentication used for requests sent to the StandardFile server.
		SetBearerToken(token string)
		// SyncItems synchronizes local items with the StandardFile server.
		SyncItems(si SyncItems) (SyncItems, error)
	}

	p      map[string]interface{}
	client struct {
		http     *http.Client
		endpoint string
		bearer   string
	}
)

// NewDefaultClient returns a new Client with default HTTP client.
func NewDefaultClient(endpoint string) (Client, error) {
	return NewClient(http.DefaultClient, endpoint)
}

// NewClient returns a new Client.
func NewClient(c *http.Client, endpoint string) (Client, error) {
	_, err := url.Parse(endpoint)
	return &client{endpoint: endpoint, http: c}, errors.Wrap(err, "could not parse endpoint")
}

func (c *client) GetAuthParams(email string) (Auth, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = "/auth/params"

	query := url.Values{}
	query.Set("email", email)
	u.RawQuery = query.Encode()

	//
	// Build request
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	//
	// Perform request
	res, err := c.http.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform request")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return nil, parseSFError(res.Body, res.StatusCode)
	}

	//
	// Process response
	var auth auth
	dec := json.NewDecoder(res.Body)
	return &auth, errors.Wrap(dec.Decode(&auth), "could not parse response")
}

func (c *client) Login(email, password string) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = "/auth/sign_in"

	//
	// Build request
	body, err := json.Marshal(p{"email": email, "password": password})
	if err != nil {
		return errors.Wrap(err, "could not serialize email & password")
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	//
	// Perform request
	res, err := c.http.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not perform request")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return parseSFError(res.Body, res.StatusCode)
	}

	//
	// Process response
	var login struct {
		Token string `json:"token"`
	}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&login)

	c.bearer = login.Token
	return errors.Wrap(err, "could not parse response")
}

func (c *client) BearerToken() string {
	return c.bearer
}

func (c *client) SetBearerToken(token string) {
	c.bearer = token
}

func (c *client) SyncItems(items SyncItems) (SyncItems, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return items, errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = "/items/sync"

	//
	// Build request
	body, err := json.Marshal(&items)
	if err != nil {
		return items, errors.Wrap(err, "could not serialize sync data")
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return items, errors.Wrap(err, "could not build request")
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.bearer))

	//
	// Perform request
	res, err := c.http.Do(req)
	if err != nil {
		return items, errors.Wrap(err, "could not perform request")
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return items, parseSFError(res.Body, res.StatusCode)
	}

	//
	// Process response
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&items)

	return items, errors.Wrap(err, "could not parse response")
}
