package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/pkg/errors"
)

const (
	HeaderBearer    = "BEARER"
	HeaderAuth      = "Authorization"
	HeaderRequestID = "X-Request-ID"
	HeaderToken     = "token"
)

type Client struct {
	URL        string       // The location of the server, for example  "http://localhost:8065"
	APIURL     string       // The api location of the server, for example "http://localhost:8065/api/v4"
	HTTPClient *http.Client // The http client
	AuthToken  string
	AuthType   string
	HTTPHeader map[string]string // Headers to be copied over for each request
}

type Response struct {
	StatusCode int
	Error      error
	RequestID  string
	Header     http.Header
}

func NewAPIClient(url string) *Client {
	url = strings.TrimRight(url, "/")
	return &Client{url, url + APIURLSuffix, &http.Client{}, "", "", map[string]string{}}
}

func (c *Client) SetToken(token string) {
	c.AuthToken = token
	c.AuthType = HeaderBearer
}

func (c *Client) getUsersRoute() string {
	return "/users"
}

func (c *Client) getUserRoute(userId string) string {
	return fmt.Sprintf(c.getUsersRoute()+"/%v", userId)
}

func (c *Client) getUsersPageRoute(page, perPage int) string {
	return fmt.Sprintf(c.getUsersRoute()+"?page=%d&per_page=%d", page, perPage)
}

func (c *Client) doApiGet(url string) (*http.Response, error) {
	return c.doApiRequest(http.MethodGet, c.APIURL+url, "")
}

func (c *Client) doApiPost(url string, data string) (*http.Response, error) {
	return c.doApiRequest(http.MethodPost, c.APIURL+url, data)
}

func (c *Client) doApiPut(url string, data string) (*http.Response, error) {
	return c.doApiRequest(http.MethodPut, c.APIURL+url, data)
}

func (c *Client) doApiDelete(url string) (*http.Response, error) {
	return c.doApiRequest(http.MethodDelete, c.APIURL+url, "")
}

func (c *Client) doApiRequest(method, url, data string) (*http.Response, error) {
	return c.doApiRequestReader(method, url, strings.NewReader(data), map[string]string{})
}

func (c *Client) doApiRequestWithHeaders(method, url, data string, headers map[string]string) (*http.Response, error) {
	return c.doApiRequestReader(method, url, strings.NewReader(data), headers)
}

func (c *Client) doApiRequestReader(method, url string, data io.Reader, headers map[string]string) (*http.Response, error) {
	rq, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, errors.Wrap(err, "can't create new request")
	}

	for k, v := range headers {
		rq.Header.Set(k, v)
	}

	if c.AuthToken != "" {
		rq.Header.Set(HeaderAuth, c.AuthType+" "+c.AuthToken)
	}

	if c.HTTPHeader != nil && len(c.HTTPHeader) > 0 {
		for k, v := range c.HTTPHeader {
			rq.Header.Set(k, v)
		}
	}

	rp, err := c.HTTPClient.Do(rq)
	if err != nil {
		return nil, errors.Wrap(err, "can't do the request")
	}
	if rp == nil {
		return nil, errors.New("nil response")
	}

	if rp.StatusCode == http.StatusNotModified {
		return rp, nil
	}

	if rp.StatusCode >= 300 {
		defer closeBody(rp)
		return rp, errorFromReader(rp.Body)
	}

	return rp, nil
}

func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = io.Copy(ioutil.Discard, r.Body)
		_ = r.Body.Close()
	}
}

func errorFromReader(data io.Reader) error {
	str := ""
	bytes, err := ioutil.ReadAll(data)
	if err != nil {
		str = err.Error()
	} else {
		str = string(bytes)
	}
	return errors.New(str)
}

func buildErrorResponse(r *http.Response, err error) *Response {
	var statusCode int
	var header http.Header
	if r != nil {
		statusCode = r.StatusCode
		header = r.Header
	} else {
		statusCode = 0
		header = make(http.Header)
	}

	return &Response{
		StatusCode: statusCode,
		Error:      err,
		Header:     header,
	}
}

func buildResponse(r *http.Response) *Response {
	return &Response{
		StatusCode: r.StatusCode,
		RequestID:  r.Header.Get(HeaderRequestID),
		Header:     r.Header,
	}
}

// GetUser returns a user based on the provided user id string.
func (c *Client) GetUser(userID string) (*model.User, *Response) {
	r, err := c.doApiGet(c.getUserRoute(userID))
	if err != nil {
		return nil, buildErrorResponse(r, err)
	}
	defer closeBody(r)
	var user *model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, &Response{
			StatusCode: 0,
			Error:      err,
			Header:     make(http.Header),
		}
	}
	return user, buildResponse(r)
}

// GetUser returns a user based on the provided user id string.
func (c *Client) GetUsers(page, perPage int) ([]*model.User, *Response) {
	r, err := c.doApiGet(c.getUsersPageRoute(page, perPage))
	if err != nil {
		return nil, buildErrorResponse(r, err)
	}
	defer closeBody(r)
	var users []*model.User
	if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
		return nil, &Response{
			StatusCode: 0,
			Error:      err,
			Header:     make(http.Header),
		}
	}
	return users, buildResponse(r)
}

// CreateUser creates a user in the system based on the provided user struct.
func (c *Client) CreateUser(user *model.User) (*model.User, *Response) {
	b, err := json.Marshal(user)
	if err != nil {
		return nil, &Response{
			StatusCode: 0,
			Error:      err,
			Header:     make(http.Header),
		}
	}
	r, err := c.doApiPost(c.getUsersRoute(), string(b))
	if err != nil {
		return nil, buildErrorResponse(r, err)
	}
	defer closeBody(r)
	var updatedUser *model.User
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		return nil, &Response{
			StatusCode: 0,
			Error:      err,
			Header:     make(http.Header),
		}
	}
	return updatedUser, buildResponse(r)
}

// LoginByUserID authenticates a user by user id and password.
func (c *Client) LoginByUserID(id string, password string) (*model.User, *Response) {
	m := make(map[string]string)
	m["user_id"] = id
	m["password"] = password
	return c.login(m)
}

// LoginByUsername authenticates a user by username and password.
func (c *Client) LoginByUsername(username string, password string) (*model.User, *Response) {
	m := make(map[string]string)
	m["username"] = username
	m["password"] = password
	return c.login(m)
}

func (c *Client) login(m map[string]string) (*model.User, *Response) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, &Response{
			StatusCode: 0,
			Error:      err,
			Header:     make(http.Header),
		}
	}
	r, err := c.doApiPost("/users/login", string(b))
	if err != nil {
		return nil, buildErrorResponse(r, err)
	}
	defer closeBody(r)
	c.AuthToken = r.Header.Get(HeaderToken)
	c.AuthType = HeaderBearer
	var user *model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, &Response{
			StatusCode: 0,
			Error:      err,
			Header:     make(http.Header),
		}
	}
	return user, buildResponse(r)
}
