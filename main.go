package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"regexp"
	"time"
	"bytes"
	"encoding/json"
	"github.com/labstack/echo/v4"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/microcosm-cc/bluemonday"
)

const LETTERS = "abcdefghijklmnopqrstuvwxyz"

type BlurbTemplates struct {
	viewHtml   *template.Template
}

func GetTemplates() BlurbTemplates {
	viewHtml, _ := template.ParseFiles("static/view.html")
	return BlurbTemplates{viewHtml}
}

type BlurbServer struct {
	BlurbTemplates
	PubSub
	BlurbDB
}

func run(port int, dbName string) {
	bs := BlurbServer{GetTemplates(), GetPubSub(), GetBlurbDB("blurbs.sqlite", "queries.sql")}
	now := time.Now()
	rand.Seed(now.UnixNano())

	e := echo.New()
	e.Static("/static", "static")
	e.GET("/", bs.getRoot)
	e.GET("/stream/:blurb", bs.getStreamingUpdates)
	e.GET("/raw/:blurb", bs.getRaw)
	e.GET("/blurb/:blurb", bs.getView)
	e.PUT("/blurb/:blurb", bs.putBlurb)
	e.Start(fmt.Sprintf(":%d", port))
	// defer bs.Close() FIXME: close the sqlite connection
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

type BlurbData struct {
	SanitizedBlurbVersion
	Png  string
}

type UnsanitizedBlurbVersion struct {
	Id   string `param:"blurb"`
	Version int64 `json:"blurb_version"`
	Body string `json:"blurb_text"`
}

type SanitizedBlurbVersion struct {
	Id   string `param:"blurb"`
	Version int64 `json:"blurb_version"`
	Body string `json:"blurb_text"`
}

func (sbv SanitizedBlurbVersion) AsInnerHTML() template.HTML {
	return template.HTML(sbv.Body)
}

func (sbv SanitizedBlurbVersion) toBytes() []byte {
	ret, err := json.Marshal(sbv)
	if err != nil {
		panic(err)
	}
	return ret
}

func (bs BlurbServer) AsFullHTML(sbv SanitizedBlurbVersion) string {
	blurbData := BlurbData{sbv,
		readPng(sbv.Id)}
	ret := bytes.Buffer{}
	bs.viewHtml.Execute(&ret, blurbData)
	return ret.String()
}

func (ubv UnsanitizedBlurbVersion) Sanitize() SanitizedBlurbVersion {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").Globally()
	p.AllowStyles("background-color", "color").MatchingHandler(rgbOfInts).Globally()
	return SanitizedBlurbVersion{ubv.Id, ubv.Version, p.Sanitize(ubv.Body)}
}


func (bs BlurbServer) GetBlurbById(blurbId string) SanitizedBlurbVersion {
	args := map[string]interface{}{
		"id": blurbId, 
	}
	var sbv SanitizedBlurbVersion


	stmt, err := bs.db.PrepareNamed(bs.queries["get_blurb"])
	if err != nil {
		println(err)
		panic(err)
	}
	err = stmt.Get(&sbv, args)
	if err != nil {
		println(err)
		panic(err)
	}
	println(sbv.Version)
	return sbv
}

func (bs BlurbServer) SaveBlurb(sbv SanitizedBlurbVersion) error {
	_, err := bs.db.NamedExec(bs.queries["put_blurb"], &sbv)
	
	if err != nil {
		println(err)
		panic(err)
	}
	return err
}
