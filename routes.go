package main

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
)

func (bs BlurbServer) getBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		template.HTML(bs.getBlurbText(blurbId)),
		getPng(blurbId)}
	ret := bytes.Buffer{}
	bs.viewHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func (bs BlurbServer) getEditor(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		template.HTML(bs.getBlurbText(blurbId)),
		""}
	ret := bytes.Buffer{}
	bs.editorHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func (bs BlurbServer) streamUpdates(c echo.Context) error {
	blurbId := c.Param("blurb")
	oldText := ""
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-control", "no-cache")
	ch, subId := bs.sub(blurbId)
	defer bs.unsub(blurbId, subId)
	for <-ch {
		text := bs.getBlurbText(blurbId)
		if text != oldText {
			for _, chunk := range strings.Split(text, "\n") {
				c.Response().Write([]byte("data: " + chunk + "\n"))
			}
			oldText = text
			c.Response().Write([]byte("\n"))
			c.Response().Flush()
		}
	}
	return nil
}

func (bs BlurbServer) getRoot(c echo.Context) error {
	c.Response().Header().Set("Cache-control", "no-cache")
	return c.Redirect(301, "/blurb/"+getNewBlurbId())
}

func (bs BlurbServer) getRaw(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := bs.getBlurbText(blurbId)
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
	err := bs.putBlurbText(blurbId, text)
	if err == nil {
		return c.Redirect(http.StatusSeeOther, "/blurb/"+blurbId)
	} else {
		return c.String(400, err.Error())
	}
}
