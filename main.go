package main

import (
	"encoding/base64"
	"html/template"
	"math/rand"
	"regexp"
	"time"

	"github.com/boltdb/bolt"
	"github.com/labstack/echo/v4"
	qrcode "github.com/skip2/go-qrcode"
)

const LETTERS = "abcdefghijklmnopqrstuvwxyz"
const METABLURB = `<u><em style='background-color: rgb(255, 240, 201)'>Blurb.cloud</em></u> is a shared, local billboard. Take an old tablet or smartphone, mount it on the wall, and browse to this page to display the blurb. Anyone with the link can change the blurb, and anyone who passes by the display can see the link.`

type BlurbTemplates struct {
	editorHtml *template.Template
	viewHtml   *template.Template
}

type BlurbData struct {
	Id   string
	Text template.HTML
	Png  string
}

func GetTemplates() BlurbTemplates {
	editorHtml, _ := template.ParseFiles("static/editor.html")
	viewHtml, _ := template.ParseFiles("static/view.html")
	return BlurbTemplates{editorHtml, viewHtml}
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
	e.GET("/stream/:blurb", cs.streamUpdates)
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

func rgbOfInts(s string) bool {
	m, _ := regexp.Match("^ ?rgb\\( ?\\d+, ?\\d+, ?\\d+\\);?$", []byte(s))
	return m
}
