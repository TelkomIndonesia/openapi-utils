package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
)

type GenReqIn struct {
	Path   GenReqInPath
	Query  GenReqInQuery
	Header GenReqInHeader
}

type GenReqInPath struct {
	Gen0Name string
}
type GenReqInQuery struct {
	Gen0Name string
}
type GenReqInHeader struct {
	Gen0Name string
}

type GenReqOut struct {
	Path   GenReqOutPath
	Query  GenReqOutQuery
	Header GenReqOutHeader
}
type GenReqOutPath struct {
	Gen0Name string
	Gen1Name string
}
type GenReqOutQuery struct {
	Gen0Name string
	Gen1Name string
}
type GenReqOutHeader struct {
	Gen0Name string
	Gen1Name string
}

type ProxyInterface interface {
	GenOp(c echo.Context, in GenReqIn) (out GenReqOut, err error)
}

type ProxyInterfaceWrapper struct {
	pi ProxyInterface

	gen0ProxyName *httputil.ReverseProxy
}

func (pw ProxyInterfaceWrapper) NameReq(c echo.Context) error {
	in := GenReqIn{}
	out, err := pw.pi.GenOp(c, in)
	if err != nil {
		return err
	}

	u := &url.URL{}
	u.Path = "/genpart1/" + out.Path.Gen0Name + "/genpart1/" + out.Path.Gen0Name + "/genpartN"
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

func Register(e *echo.Echo, pi ProxyInterface) (err error) {
	genUrl1, err := url.Parse("https://api/")
	if err != nil {
		return err
	}

	pw := ProxyInterfaceWrapper{
		pi: pi,

		gen0ProxyName: httputil.NewSingleHostReverseProxy(genUrl1),
	}

	e.Any("genpath", pw.NameReq)

	return
}

// implementation
var _ ProxyInterface = proxyimplementation{}

type proxyimplementation struct{}

// GenOp implements ProxyInterface.
func (p proxyimplementation) GenOp(c echo.Context, in GenReqIn) (out GenReqOut, err error) {
	panic("unimplemented")
}

func main() {
	e := echo.New()
	err := Register(e, proxyimplementation{})
	if err != nil {
		log.Fatalln(err)
	}
	log.Fatalln(e.Start(":80"))
}
