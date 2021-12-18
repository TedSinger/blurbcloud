package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	qrcode "github.com/skip2/go-qrcode"
)

const LETTERS = "abcdefghijklmnopqrstuvwxyz"
const METABLURB = `<u><em style='background-color: rgb(255, 240, 201)'>Blurb.cloud</em></u> is a shared, local billboard. Anyone who sees a blurb can change the blurb.`

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
	BlurbDB
	BlurbTemplates
	PubSub
}

func run(port int, dbName string) {
	bs := BlurbServer{GetBlurbDB(dbName), GetTemplates(), GetPubSub()}
	now := time.Now()
	rand.Seed(now.UnixNano())

	e := echo.New()
	e.Static("/static", "static")
	e.GET("/", bs.getRoot)
	e.GET("/stream/:blurb", bs.getStreamingUpdates)
	e.GET("/editor/:blurb", bs.getEditor)
	e.GET("/raw/:blurb", bs.getRaw)
	e.GET("/blurb/:blurb", bs.getView)
	// https://softwareengineering.stackexchange.com/questions/114156/why-are-there-are-no-put-and-delete-methods-on-html-forms
	e.POST("/blurb/:blurb", bs.putBlurb)
	e.Start(fmt.Sprintf(":%d", port))
	defer bs.Close()
}

func main() {
	port := flag.Int("port", 22000, "")
	dbName := flag.String("db", "blurbs.db", "boltdb filename")
	flag.Parse()
	run(*port, *dbName)
}

func genNewBlurbId() string {
	ret := ""
	for i := 0; i < 4; i++ {
		n := rand.Intn(len(LETTERS))
		ret += LETTERS[n : n+1]
	}

	return ret
}

func readPng(blurbId string) string {
	var png []byte
	// should i cache any of these in the db?
	png, err := qrcode.Encode("http://blurb.cloud/blurb/"+blurbId, qrcode.Low, 120)
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
