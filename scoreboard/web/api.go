// Copyright(C) 2020 iDigitalFlame
//
// This program is free software: you can redistribute it and / or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.If not, see <https://www.gnu.org/licenses/>.
//

package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"
)

var (
	// ErrInvalidTimeout is returned by 'NewAPI' when the given timeout is less than zero.
	ErrInvalidTimeout = errors.New("timeout must be greater than or equal to zero")
)

// API is a struct that represents an API caller.
// This struct allows for polling and getting data from a API endpoint in the form of bytes.
// Also supports JSON data calling.
type API struct {
	client  *http.Client
	headers map[string]string
	timeout time.Duration

	url.URL
}

// Get attempts a HTTP GET request at the provided URL + the base URL. This will return an error
// on non 200 HTTP status codes or if a timeout occurs. If successful, the binary data in a byte array will
// be returned.
func (a API) Get(urlpath string) ([]byte, error) {
	a.Path = fmt.Sprintf("%s/", path.Join(a.Path, urlpath))
	r, err := http.NewRequest(http.MethodGet, a.String(), nil)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	defer o.Body.Close()
	if o.StatusCode >= 400 {
		return nil, fmt.Errorf("request \"%s\" returned status code \"%d\"", a.String(), o.StatusCode)
	}
	b, err := ioutil.ReadAll(o.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading from URL \"%s\": %w", a.String(), err)
	}
	return b, nil
}

// GetJSON is similar to the 'Get' function, but instead will attempt to unmarshal the provided
// binary data into the supplied object 'obj'. The function will return an error if the JSON
// is not formatted correctly or a HTTP or timeout error occurs.
func (a API) GetJSON(urlpath string, obj interface{}) error {
	r, err := a.Get(urlpath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, &obj); err != nil {
		return fmt.Errorf("unable to unmarshal JSON: %w", err)
	}
	return nil
}

// Post attempts a HTTP POST request at the provided URL + the base URL. The data Posted to the
// server will be contained in provided io.Reader. This will return an error
// if a timeout occurs. If successful, the binary data in a byte array will
// be returned.
func (a API) Post(urlpath string, data io.Reader) ([]byte, error) {
	a.Path = fmt.Sprintf("%s/", path.Join(a.Path, urlpath))
	r, err := http.NewRequest(http.MethodPost, a.String(), data)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	defer o.Body.Close()
	if o.StatusCode >= 400 {
		return nil, fmt.Errorf("request \"%s\" returned status code \"%d\"", a.String(), o.StatusCode)
	}
	b, err := ioutil.ReadAll(o.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading from URL \"%s\": %w", a.String(), err)
	}
	return b, nil
}

// PostJSON is similar to the 'Post' function, but instead will attempt to unmarshal the provided
// binary data into the supplied object 'obj'. The function will return an error if the JSON
// is not formatted correctly or a HTTP or timeout error occurs.
func (a API) PostJSON(urlpath string, data io.Reader, obj interface{}) error {
	r, err := a.Post(urlpath, data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, &obj); err != nil {
		return fmt.Errorf("unable to unmarshal JSON: %w", err)
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
		return nil, fmt.Errorf("could not unpack provided URL \"%s\": %w", baseurl, err)
	}
	if !u.IsAbs() {
		u.Scheme = "http"
	}
	a := &API{
		URL:     *u,
		headers: headers,
		timeout: timeout,
	}
	if timeout > 0 {
		a.client = &http.Client{
			Timeout: a.timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   a.timeout,
					KeepAlive: a.timeout,
					DualStack: true,
				}).DialContext,
				IdleConnTimeout:       a.timeout,
				TLSHandshakeTimeout:   a.timeout,
				ExpectContinueTimeout: a.timeout,
				ResponseHeaderTimeout: a.timeout,
			},
		}
	} else {
		a.client = &http.Client{
			Transport: &http.Transport{
				Proxy:       http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{DualStack: true}).DialContext,
			},
		}
	}
	return a, nil
}
