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
const METABLURB = `<u><em style='background-color: rgb(255, 240, 201)'>Blurb.cloud</em></u> is a shared, local billboard. Anyone who sees a blurb can change the blurb.`

type BlurbTemplates struct {
	viewHtml   *template.Template
}

func GetTemplates() BlurbTemplates {
	viewHtml, _ := template.ParseFiles("static/view.html")
	return BlurbTemplates{viewHtml}
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
	e.GET("/raw/:blurb", bs.getRaw)
	e.GET("/blurb/:blurb", bs.getView)
	e.PUT("/blurb/:blurb", bs.putBlurb)
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

type BlurbData struct {
	BlurbVersion
	Png  string
}

type BlurbVersion struct {
	Id   string
	Version int
	Text template.HTML
}

func DefaultBlurbVersion(id string) BlurbVersion {
	return BlurbVersion{id, 0, METABLURB}
}

func BlurbVersionFromJson(data []byte) BlurbVersion {
	ubv := new(UnsanitizedBlurbVersion)
	err := json.Unmarshal(data, ubv)
	if err != nil {
		panic(err)
	}
	return BlurbVersion{ubv.Id, ubv.Version, template.HTML(ubv.Text)}
}

type UnsanitizedBlurbVersion struct {
	Id   string `param:"blurb"`
	Version int `json:"blurb_version"`
	Text string `json:"blurb_text"`
}

type SanitizedBlurbVersion struct {
	Id   string `param:"blurb"`
	Version int `json:"blurb_version"`
	Text string `json:"blurb_text"`
}

func (sbv SanitizedBlurbVersion) toBytes() []byte {
	ret, err := json.Marshal(sbv)
	if err != nil {
		panic(err)
	}
	return ret
}

func (ubv UnsanitizedBlurbVersion) Sanitize() SanitizedBlurbVersion {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").Globally()
	p.AllowStyles("background-color", "color").MatchingHandler(rgbOfInts).Globally()
	return SanitizedBlurbVersion{ubv.Id, ubv.Version, p.Sanitize(ubv.Text)}
}

func (bs BlurbServer) BlurbHTML(blurbId string) string {
	data := bs.readBlurb(blurbId)
	var bv BlurbVersion
	if data == nil {
		bv = DefaultBlurbVersion(blurbId)
	} else {
		bv = BlurbVersionFromJson(data)	
	}

	blurbData := BlurbData{bv,
		readPng(blurbId)}
	ret := bytes.Buffer{}
	bs.viewHtml.Execute(&ret, blurbData)
	return ret.String()
}

func (bs BlurbServer) SaveBlurb(sbv SanitizedBlurbVersion) error {
	data := bs.readBlurb(sbv.Id)
	var bv BlurbVersion
	if data == nil {
		bv = DefaultBlurbVersion(sbv.Id)
	} else {
		bv = BlurbVersionFromJson(data)	
	}
	if bv.Version < sbv.Version {
		return bs.writeBlurb(sbv.Id, sbv.toBytes())	
	} else {
		return nil
	}
}