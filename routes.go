package main

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
)

type BlurbData struct {
	Id   string
	Text template.HTML
	Png  string
}

func (bs BlurbServer) getView(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		template.HTML(bs.readBlurb(blurbId)),
		readPng(blurbId)}
	ret := bytes.Buffer{}
	bs.viewHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func (bs BlurbServer) getEditor(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		template.HTML(bs.readBlurb(blurbId)),
		""}
	ret := bytes.Buffer{}
	bs.editorHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func (bs BlurbServer) getRoot(c echo.Context) error {
	c.Response().Header().Set("Cache-control", "no-cache")
	return c.Redirect(301, "/blurb/"+genNewBlurbId())
}

func (bs BlurbServer) getRaw(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := bs.readBlurb(blurbId)
	return c.String(200, text)
}

func (bs BlurbServer) putBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := c.FormValue("text")
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").Globally()
	p.AllowStyles("background-color", "color").MatchingHandler(rgbOfInts).Globally()
	text = p.Sanitize(text)
	println(blurbId + " : " + text)
	err := bs.writeBlurb(blurbId, text)
	if err == nil {
		go bs.pub(blurbId, text)
		return c.Redirect(http.StatusSeeOther, "/blurb/"+blurbId)
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
	for text := range ch {
		if text != oldText {
			for _, chunk := range strings.Split(text, "\n") {
				c.Response().Write([]byte("data: " + chunk + "\n"))
			}
			c.Response().Write([]byte("\n"))
			c.Response().Flush()
			oldText = text
		}
	}
	return nil
}
