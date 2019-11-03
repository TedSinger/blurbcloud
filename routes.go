package main

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
)

func (cs BlurbServer) getBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		template.HTML(cs.getBlurbText(blurbId)),
		getPng(blurbId)}
	ret := bytes.Buffer{}
	cs.viewHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func (cs BlurbServer) getEditor(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		template.HTML(cs.getBlurbText(blurbId)),
		""}
	ret := bytes.Buffer{}
	cs.editorHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func (cs BlurbServer) streamUpdates(c echo.Context) error {
	blurbId := c.Param("blurb")
	oldText := ""
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-control", "no-cache")
	ch, subId := cs.sub(blurbId)
	defer cs.unsub(blurbId, subId)
	for <-ch {
		text := cs.getBlurbText(blurbId)
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

func (cs BlurbServer) getRoot(c echo.Context) error {
	c.Response().Header().Set("Cache-control", "no-cache")
	return c.Redirect(301, "/blurb/"+getNewBlurbId())
}

func (cs BlurbServer) getRaw(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := cs.getBlurbText(blurbId)
	return c.String(200, text)
}

func (cs BlurbServer) putBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := c.FormValue("text")
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").Globally()
	p.AllowStyles("background-color", "color").MatchingHandler(rgbOfInts).Globally()
	text = p.Sanitize(text)
	println(blurbId + " : " + text)
	err := cs.putBlurbText(blurbId, text)
	if err == nil {
		return c.Redirect(http.StatusSeeOther, "/blurb/"+blurbId)
	} else {
		return c.String(400, err.Error())
	}
}