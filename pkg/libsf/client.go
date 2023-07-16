package libsf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
)

type (
	// A Client defines all interactions that can be performed on a StandardFile server.
	Client interface {
		// GetAuthParams returns the parameters of a user from StandardFile server.
		GetAuthParams(email string) (Auth, error)
		// Login connects the Client to the StandardFile server.
		Login(email, password string) error
		// Logout disconnects the client (since API 20200115).
		Logout() error
		// BearerToken returns the authentication used for requests sent to the StandardFile server.
		// It can be a JWT or an access token from a session.
		BearerToken() string
		// SetBearerToken sets the authentication used for requests sent to the StandardFile server.
		// It can be a JWT or an access token from a session.
		SetBearerToken(token string)
		// Session returns the authentication session used for authentication (since API 20200115).
		Session() Session
		// SetSession sets the authentication session used for authentication (since API 20200115).
		// It also uses its access token as the bearer token.
		SetSession(session Session)
		// RefreshSession gets a new pair of tokens by refreshing the session.
		RefreshSession(access, refresh string) (*Session, error)
		// SyncItems synchronizes local items with the StandardFile server.
		SyncItems(si SyncItems) (SyncItems, error)
	}

	p      map[string]any
	client struct {
		http       *http.Client
		apiversion string
		endpoint   string
		bearer     string
		session    Session
	}
)

// NewDefaultClient returns a new Client with default HTTP client.
func NewDefaultClient(endpoint string) (Client, error) {
	return NewClient(http.DefaultClient, APIVersion, endpoint)
}

// NewClient returns a new Client.
func NewClient(c *http.Client, apiversion string, endpoint string) (Client, error) {
	_, err := url.Parse(endpoint)
	return &client{apiversion: apiversion, endpoint: endpoint, http: c}, errors.Wrap(err, "could not parse endpoint")
}

func (c *client) GetAuthParams(email string) (Auth, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = path.Join(u.Path, "/auth/params")

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
	u.Path = path.Join(u.Path, "/auth/sign_in")

	//
	// Build request
	body, err := json.Marshal(p{"api": c.apiversion, "email": email, "password": password})
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
		Token   string  `json:"token"`   // JWT before 20200115
		Session Session `json:"session"` // Since 20200115
	}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&login)

	c.bearer = login.Token
	if login.Session.Defined() {
		c.SetSession(login.Session)
	}
	return errors.Wrap(err, "could not parse response")
}

func (c *client) Logout() error {
	if !c.session.Defined() {
		return errors.New("no session defined")
	}

	//

	u, err := url.Parse(c.endpoint)
	if err != nil {
		return errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = path.Join(u.Path, "/auth/sign_out")

	//
	// Build request
	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.bearer))

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

	return nil
}

func (c *client) BearerToken() string {
	return c.bearer
}

func (c *client) SetBearerToken(token string) {
	c.bearer = token
}

func (c *client) Session() Session {
	return c.session
}

func (c *client) SetSession(session Session) {
	c.session = session
	c.bearer = c.session.AccessToken
}

func (c *client) RefreshSession(access, refresh string) (*Session, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = path.Join(u.Path, "/session/refresh")

	//
	// Build request
	body, err := json.Marshal(p{"access_token": access, "refresh_token": refresh})
	if err != nil {
		return nil, errors.Wrap(err, "could not serialize refresh session data")
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.bearer))

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
	var session = struct {
		Session Session `json:"session"`
	}{}

	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&session)

	return &session.Session, errors.Wrap(err, "could not parse response")
}

func (c *client) SyncItems(items SyncItems) (SyncItems, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return items, errors.Wrap(err, "could not parse endpoint")
	}
	u.Path = path.Join(u.Path, "/items/sync")

	//
	// Build request
	items.API = c.apiversion
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
