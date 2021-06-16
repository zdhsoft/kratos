package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
	"github.com/gorilla/mux"
)

var _ Context = (*wrapper)(nil)

// HandlerFunc defines a function to serve HTTP requests.
type HandlerFunc func(Context) error

// Context is an HTTP Context.
type Context interface {
	context.Context
	Vars() url.Values
	Query() url.Values
	Form() url.Values
	Header() http.Header
	Request() *http.Request
	Response() http.ResponseWriter
	Middleware(middleware.Handler) middleware.Handler
	Bind(interface{}) error
	BindVars(interface{}) error
	BindQuery(interface{}) error
	BindForm(interface{}) error
	Returns(interface{}, error) error
	Result(int, interface{}) error
	JSON(int, interface{}) error
	XML(int, interface{}) error
	String(int, string) error
	Blob(int, string, []byte) error
	Stream(int, string, io.Reader) error
	Reset(http.ResponseWriter, *http.Request)
}

type wrapper struct {
	route *Route
	req   *http.Request
	res   http.ResponseWriter
}

func (c *wrapper) Header() http.Header {
	return c.req.Header
}

func (c *wrapper) Vars() url.Values {
	raws := mux.Vars(c.req)
	vars := make(url.Values, len(raws))
	for k, v := range raws {
		vars[k] = []string{v}
	}
	return vars
}
func (c *wrapper) Form() url.Values {
	if err := c.req.ParseForm(); err != nil {
		return url.Values{}
	}
	return c.req.Form
}
func (c *wrapper) Query() url.Values {
	return c.req.URL.Query()
}
func (c *wrapper) Request() *http.Request        { return c.req }
func (c *wrapper) Response() http.ResponseWriter { return c.res }
func (c *wrapper) Middleware(h middleware.Handler) middleware.Handler {
	return middleware.Chain(c.route.srv.ms...)(h)
}
func (c *wrapper) Bind(v interface{}) error      { return c.route.srv.dec(c.req, v) }
func (c *wrapper) BindVars(v interface{}) error  { return binding.BindQuery(c.Vars(), v) }
func (c *wrapper) BindQuery(v interface{}) error { return binding.BindQuery(c.Query(), v) }
func (c *wrapper) BindForm(v interface{}) error  { return binding.BindForm(c.req, v) }
func (c *wrapper) Returns(v interface{}, err error) error {
	if err != nil {
		return err
	}
	if err := c.route.srv.enc(c.res, c.req, v); err != nil {
		return err
	}
	return nil
}
func (c *wrapper) Result(code int, v interface{}) error {
	c.res.WriteHeader(code)
	if err := c.route.srv.enc(c.res, c.req, v); err != nil {
		return err
	}
	return nil
}
func (c *wrapper) JSON(code int, v interface{}) error {
	c.res.WriteHeader(code)
	c.res.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.res).Encode(v)
}
func (c *wrapper) XML(code int, v interface{}) error {
	c.res.WriteHeader(code)
	c.res.Header().Set("Content-Type", "application/xml")
	return xml.NewEncoder(c.res).Encode(v)
}
func (c *wrapper) String(code int, text string) error {
	c.res.WriteHeader(code)
	c.res.Header().Set("Content-Type", "text/plain")
	c.res.Write([]byte(text))
	return nil
}
func (c *wrapper) Blob(code int, contentType string, data []byte) error {
	c.res.WriteHeader(code)
	c.res.Header().Set("Content-Type", contentType)
	c.res.Write(data)
	return nil
}
func (c *wrapper) Stream(code int, contentType string, rd io.Reader) error {
	c.res.WriteHeader(code)
	c.res.Header().Set("Content-Type", contentType)
	_, err := io.Copy(c.res, rd)
	return err
}
func (c *wrapper) Reset(res http.ResponseWriter, req *http.Request) {
	c.res = res
	c.req = req
}
func (c *wrapper) Deadline() (time.Time, bool) {
	if c.req == nil {
		return time.Time{}, false
	}
	return c.req.Context().Deadline()
}
func (c *wrapper) Done() <-chan struct{} {
	if c.req == nil {
		return nil
	}
	return c.req.Context().Done()
}
func (c *wrapper) Err() error {
	if c.req == nil {
		return context.Canceled
	}
	return c.req.Context().Err()
}
func (c *wrapper) Value(key interface{}) interface{} {
	if c.req == nil {
		return nil
	}
	return c.req.Context().Value(key)
}