package main

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)



func (bs BlurbServer) getView(c echo.Context) error {
	blurbId := c.Param("blurb")
	sbv := bs.GetBlurbById(blurbId)
	return c.HTML(http.StatusOK, bs.AsFullHTML(sbv))
}

func (bs BlurbServer) getRoot(c echo.Context) error {
	c.Response().Header().Set("Cache-control", "no-cache")
	return c.Redirect(301, "/blurb/"+genNewBlurbId())
}

func (bs BlurbServer) getRaw(c echo.Context) error {
	blurbId := c.Param("blurb")
	sbv := bs.GetBlurbById(blurbId)
	return c.String(200, bs.AsFullHTML(sbv))
}

func (bs BlurbServer) putBlurb(c echo.Context) error {
	ubv := new(UnsanitizedBlurbVersion)
	if err := c.Bind(ubv); err != nil {
		println(err)
		panic(err)
	}
	sbv := ubv.Sanitize()

	err := bs.SaveBlurb(sbv)
	if err == nil {
		go bs.pub(sbv)
		return c.String(200, "OK")
	} else {
		return c.String(400, err.Error())
	}
}

func (bs BlurbServer) getStreamingUpdates(c echo.Context) error {
	blurbId := c.Param("blurb")
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-control", "no-cache")
	ch, subId := bs.sub(blurbId)
	oldText := ""
	defer bs.unsub(blurbId, subId) // this does not work - the client closing the connection is undetectable here, and this function never terminates
	for blurb_version := range ch {
		if blurb_version.Body != oldText {
			for _, chunk := range strings.Split(string(blurb_version.toBytes()), "\n") {
				c.Response().Write([]byte("data: " + chunk + "\n"))
			}
			c.Response().Write([]byte("\n"))
			c.Response().Flush()
			oldText = blurb_version.Body
		}
	}
	return nil
}
