package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/xerrors"
)

var (
	// ErrInvalidTimeout is returned by 'NewAPI' when the given timeout is less than zero.
	ErrInvalidTimeout = xerrors.New("timeout must be greater than or equal to zero")
)

// API is a struct that repersents an API caller.
// This struct allows for polling and getting data from a API endpoint in the form of bytes.
// Also supports JSON data calling.
type API struct {
	Base *url.URL

	client  *http.Client
	headers map[string]string
	timeout time.Duration
}

// Get attempts a HTTP GET request at the provded URL + the base URL. This will return an error
// on non 200 HTTP status codes or if a timeout occurs. If sucessful, the binary data in a byte array will
// be returned.
func (a *API) Get(urlpath string) ([]byte, error) {
	x := *(a.Base)
	u := &x
	u.Path = fmt.Sprintf("%s/", path.Join(a.Base.Path, urlpath))
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, xerrors.Errorf("could not retrive URL \"%s\": %w", u.String(), err)
	}
	if a.timeout > 0 {
		x, c := context.WithTimeout(context.Background(), a.timeout)
		r = r.WithContext(x)
		defer c()
	}
	if a.headers != nil && len(a.headers) > 0 {
		for k, v := range a.headers {
			r.Header.Add(k, v)
		}
	}
	o, err := a.client.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("could not retrive URL \"%s\": %w", u.String(), err)
	}
	defer o.Body.Close()
	if o.StatusCode >= 400 {
		return nil, xerrors.Errorf("request for \"%s\" returned status code \"%d\"", u.String(), o.StatusCode)
	}
	b, err := ioutil.ReadAll(o.Body)
	if err != nil {
		return nil, xerrors.Errorf("could not read data from URL \"%s\": %w", u.String(), err)
	}
	return b, nil
}

// GetJSON is similar to the 'Get' function, but instead will attempt to unmarshal the provided
// binary data into the supplied object 'obj'. The function will return an error if the JSON
// is not formatted corectly or a HTTP or timeout error occurs.
func (a *API) GetJSON(urlpath string, obj interface{}) error {
	r, err := a.Get(urlpath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, &obj); err != nil {
		return xerrors.Errorf("unable to unmarshal JSON: %w", err)
	}
	return nil
}

// Post attempts a HTTP POST request at the provded URL + the base URL. The data Posted to the
// server will be contained in provded io.Reader. This will return an error
// if a timeout occurs. If sucessful, the binary data in a byte array will
// be returned.
func (a *API) Post(urlpath string, data io.Reader) ([]byte, error) {
	x := *(a.Base)
	u := &x
	u.Path = fmt.Sprintf("%s/", path.Join(a.Base.Path, urlpath))
	r, err := http.NewRequest(http.MethodPost, u.String(), data)
	if err != nil {
		return nil, xerrors.Errorf("could not retrive URL \"%s\": %w", u.String(), err)
	}
	if a.timeout > 0 {
		x, c := context.WithTimeout(context.Background(), a.timeout)
		r = r.WithContext(x)
		defer c()
	}
	if a.headers != nil && len(a.headers) > 0 {
		for k, v := range a.headers {
			r.Header.Add(k, v)
		}
	}
	o, err := a.client.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("could not retrive URL \"%s\": %w", u.String(), err)
	}
	defer o.Body.Close()
	if o.StatusCode >= 400 {
		return nil, xerrors.Errorf("request for \"%s\" returned status code \"%d\"", u.String(), o.StatusCode)
	}
	b, err := ioutil.ReadAll(o.Body)
	if err != nil {
		return nil, xerrors.Errorf("could not read data from URL \"%s\": %w", u.String(), err)
	}
	return b, nil
}

// PostJSON is similar to the 'Post' function, but instead will attempt to unmarshal the provided
// binary data into the supplied object 'obj'. The function will return an error if the JSON
// is not formatted corectly or a HTTP or timeout error occurs.
func (a *API) PostJSON(urlpath string, data io.Reader, obj interface{}) error {
	r, err := a.Post(urlpath, data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, &obj); err != nil {
		return xerrors.Errorf("unable to unmarshal JSON: %w", err)
	}
	return nil
}

// NewAPI creates and returns a new API struct. The arguments taken are the baseurl as a string,
// the timeout value as a time.Duration value (timeout of zero is an infinate wait),
// and a string map of headers (which can be nil). The errors returned can be a URL parse error
// if the provided URL is invalid, or a invalid timeout error if timeout is negative.
func NewAPI(baseurl string, timeout time.Duration, headers map[string]string) (*API, error) {
	if timeout < 0 {
		return nil, ErrInvalidTimeout
	}
	u, err := url.Parse(baseurl)
	if err != nil {
		return nil, xerrors.Errorf("could not unpack provided baseurl \"%s\": %w", baseurl, err)
	}
	if !u.IsAbs() {
		u.Scheme = "http"
	}
	a := &API{
		Base:    u,
		headers: headers,
		timeout: timeout,
	}
	if timeout > 0 {
		a.client = &http.Client{
			Timeout: a.timeout,
			Transport: &http.Transport{
				Dial:                (&net.Dialer{Timeout: a.timeout}).Dial,
				TLSHandshakeTimeout: a.timeout,
			},
		}
	} else {
		a.client = &http.Client{}
	}
	return a, nil
}
