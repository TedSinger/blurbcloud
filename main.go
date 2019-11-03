package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	qrcode "github.com/skip2/go-qrcode"
)

const LETTERS = "abcdefghijklmnopqrstuvwxyz"
const METABLURB = `<u><em style='background-color: rgb(255, 240, 201)'>Blurb.cloud</em></u> is a shared, local billboard. Take an old tablet or smartphone, mount it on the wall, and browse to this page to display the blurb. Anyone with the link can change the blurb, and anyone who passes by the display can see the link.`

type BlurbTemplates struct {
	editorHead string
	editorJs   string
	viewHtml   *template.Template
}

type BlurbData struct {
	Id   string
	Text string
	Png  string
}

func GetTemplates() BlurbTemplates {
	editorHead, _ := ioutil.ReadFile("static/editorHead.html")
	editorJs, _ := ioutil.ReadFile("static/editorFuncs.js")
	viewHtml, _ := template.ParseFiles("static/view.html")
	return BlurbTemplates{string(editorHead), string(editorJs), viewHtml}
}

type BlurbServer struct {
	db *bolt.DB
	BlurbTemplates
}

func main() {
	db, _ := bolt.Open("blurbs.db", 0600, nil)

	cs := BlurbServer{db, GetTemplates()}
	cs.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Blurbs"))
		return nil
	})

	e := echo.New()
	e.Static("/static", "static")
	e.GET("/", cs.getRoot)
	e.GET("/stream/:blurb", cs.streamingUpdates)
	e.GET("/editor/:blurb", cs.getEditor)
	e.GET("/raw/:blurb", cs.getRaw)
	e.GET("/blurb/:blurb", cs.getBlurb)
	// https://softwareengineering.stackexchange.com/questions/114156/why-are-there-are-no-put-and-delete-methods-on-html-forms
	e.POST("/blurb/:blurb", cs.putBlurb)
	e.Start(":22000")
	defer db.Close()
}

func getNewBlurbId() string {
	now := time.Now()
	rand.Seed(now.UnixNano())
	ret := ""
	for i := 0; i < 4; i++ {
		n := rand.Intn(len(LETTERS))
		ret += LETTERS[n : n+1]
	}

	return ret
}

func (cs BlurbServer) streamingUpdates(c echo.Context) error {
	blurbId := c.Param("blurb")
	oldText := ""
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-control", "no-cache")
	for {
		text := cs.getBlurbText(blurbId)
		if text != oldText {
			for _, chunk := range strings.Split(text, "\n") {
				c.Response().Write([]byte("data: " + chunk + "\n"))
			}
			oldText = text
			c.Response().Write([]byte("\n"))
			c.Response().Flush()
		}
		// TODO: use gochannels instead of local polling
		time.Sleep(1 * time.Second)
	}
}

func (cs BlurbServer) getEditor(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := cs.getBlurbText(blurbId)
	template := `
	<html>
        %s
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<body>
		<form action="/blurb/%s" method="POST" target="_self">
			<div id="formstuff">
				<input type="submit" value="Post this Blurb"></input>
				<textarea type="text" name="text">%s</textarea>
				<div id="quill"></div>
			</div>
		</form>
		<script type="text/javascript">
			%s
			// FIXME: interacts badly with quotes in the inline styles
			initQuill("%s")
		</script>
	</body>
	</html>
	`
	return c.HTML(http.StatusOK, fmt.Sprintf(template, cs.editorHead, blurbId, text, cs.editorJs, text))
}

func (cs BlurbServer) getRoot(c echo.Context) error {
	c.Response().Header().Set("Cache-control", "no-cache")
	return c.Redirect(301, "/blurb/"+getNewBlurbId())
}

func (cs BlurbServer) getBlurbText(blurbId string) string {
	var text []byte
	cs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Blurbs"))
		text = b.Get([]byte(blurbId))
		return nil
	})
	if text == nil {
		return METABLURB
	} else {
		return string(text)
	}
}

func getPng(blurbId string) string {
	var png []byte
	png, err := qrcode.Encode("http://blurb.cloud/blurb/"+blurbId, qrcode.High, 120)
	if err != nil {
		return ""
	} else {
		return base64.StdEncoding.EncodeToString(png)
	}
}

func (cs BlurbServer) getRaw(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := cs.getBlurbText(blurbId)
	return c.String(200, text)
}

func (cs BlurbServer) getBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	blurbData := BlurbData{blurbId,
		cs.getBlurbText(blurbId),
		getPng(blurbId)}
	ret := bytes.Buffer{}
	cs.viewHtml.Execute(&ret, blurbData)
	return c.HTML(http.StatusOK, ret.String())
}

func rgbOfInts(s string) bool {
	m, _ := regexp.Match("^ ?rgb\\( ?\\d+, ?\\d+, ?\\d+\\);?$", []byte(s))
	return m
}

func (cs BlurbServer) putBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := c.FormValue("text")
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").Globally()
	p.AllowStyles("background-color", "color").MatchingHandler(rgbOfInts).Globally()
	text = p.Sanitize(text)
	println(blurbId + " : " + text)
	err := cs.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Blurbs"))
		b := tx.Bucket([]byte("Blurbs"))
		err := b.Put([]byte(blurbId), []byte(text))
		return err
	})
	if err == nil {
		return c.Redirect(http.StatusSeeOther, "/blurb/"+blurbId)
	} else {
		return c.String(400, err.Error())
	}
}
