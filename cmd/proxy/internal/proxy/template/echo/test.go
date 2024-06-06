package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
)

type Gen0ReqIn struct {
	Path   Gen0ReqInPath
	Query  Gen0ReqInQuery
	Header Gen0ReqInHeader
}

type Gen0ReqInPath struct {
	Gen0Name string
}
type Gen0ReqInQuery struct {
	Gen0Name string
}
type Gen0ReqInHeader struct {
	Gen0Name string
}

type Gen0ReqOut struct {
	Path   Gen0ReqOutPath
	Query  Gen0ReqOutQuery
	Header Gen0ReqOutHeader
}
type Gen0ReqOutPath struct {
	Gen0Name string
	Gen1Name string
}
type Gen0ReqOutQuery struct {
	Gen0Name string
	Gen1Name string
}
type Gen0ReqOutHeader struct {
	Gen0Name string
	Gen1Name string
}

type ProxyInterface interface {
	Gen0Op(c echo.Context, in Gen0ReqIn) (out Gen0ReqOut, err error)
}

type ProxyInterfaceWrapper struct {
	pi ProxyInterface

	gen0ProxyName *httputil.ReverseProxy
}

func (pw ProxyInterfaceWrapper) NameReq(c echo.Context) error {
	in := Gen0ReqIn{}
	out, err := pw.pi.Gen0Op(c, in)
	if err != nil {
		return err
	}

	u := &url.URL{}
	u.Path = "/gen0part0/" + out.Path.Gen0Name + "/gen0part1/" + out.Path.Gen1Name + "/gen0partN"
	u.Query().Set("gen0name", out.Query.Gen0Name)
	u.Query().Set("gen1name", out.Query.Gen1Name)
	h := http.Header{}
	h.Set("gen0name", out.Header.Gen0Name)
	h.Set("gen1name", out.Header.Gen1Name)

	req := c.Request()
	req.URL = u
	req.Header = h

	pw.gen0ProxyName.ServeHTTP(c.Response(), c.Request())
	return nil
}

type ProxyURLInterface interface {
	Gen0ProxyUrl() *url.URL
}

func Register(e *echo.Echo, pi ProxyInterface, purl ProxyURLInterface) (err error) {

	pw := ProxyInterfaceWrapper{
		pi: pi,

		gen0ProxyName: httputil.NewSingleHostReverseProxy(purl.Gen0ProxyUrl()),
	}

	e.Any("gen0path", pw.NameReq)

	return
}

// implementation
var _ ProxyInterface = proxyimplementation{}

type proxyimplementation struct{}

// Gen0Op implements ProxyInterface.
func (p proxyimplementation) Gen0Op(c echo.Context, in Gen0ReqIn) (out Gen0ReqOut, err error) {
	panic("unimplemented")
}

var _ ProxyURLInterface = proxyURLimplementation{}

type proxyURLimplementation struct{}

// Gen0ProxyUrl implements ProxyURLInterface.
func (p proxyURLimplementation) Gen0ProxyUrl() *url.URL {
	panic("unimplemented")
}

func main() {
	e := echo.New()
	err := Register(e, proxyimplementation{}, proxyURLimplementation{})
	if err != nil {
		log.Fatalln(err)
	}
	log.Fatalln(e.Start(":80"))
}
