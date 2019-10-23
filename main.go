package main

import (
	"github.com/labstack/echo/v4"
	"github.com/boltdb/bolt"
	"github.com/microcosm-cc/bluemonday"
	"net/http"
	"time"
	"fmt"
	"math/rand"
	"strings"
	"regexp"
)

const LETTERS = "abcdefghijklmnopqrstuvwxyz"
const METABLURB = `<u><em>Blurb.cloud</em></u> is a shared, local billboard. Take an old tablet or smartphone, mount it on the wall, and browse to this page to display the blurb. Anyone with the link can change the blurb, and anyone who passes by the display can see the link.`

type BlurbServer struct {
	db *bolt.DB
}

func main() {
	db, _ := bolt.Open("blurbs.db", 0600, nil)
	
	cs := BlurbServer{db}
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
		ret += LETTERS[n:n+1]
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
	return nil
}

func (cs BlurbServer) getEditor(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := cs.getBlurbText(blurbId)
	style := `<head>
	<link href="https://cdn.quilljs.com/1.3.6/quill.snow.css" rel="stylesheet">
	<style>.ql-editor { letter-spacing: 1px; line-height: 1.4; height: auto; }
	#quill { height: auto; }
	</style>
    <script src="https://cdn.quilljs.com/1.3.6/quill.js"></script>
	</head>`
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
			var quill = new Quill('#quill', {
				theme: 'snow',
				modules: {'toolbar':['bold', 'italic', 'underline', 'strike', {'script':'sub'}, {'script':'super'}, { 'color': [] }, { 'background': [] }]}
			});
			var textarea = document.querySelector('textarea');
			var toolbar = document.querySelector('.ql-toolbar');
			var button = document.querySelector('input');
			toolbar.insertBefore(button, toolbar.childNodes[0]);
			var editor = document.querySelector('.ql-editor');
			editor.innerHTML = textarea.value;
			textarea.style.visibility = 'hidden';
			quill.on('text-change', function(delta) {
				textarea.value = editor.innerHTML;
			});
			var formstuff = document.querySelector("#formstuff");
			var quilldiv = document.querySelector("#quill")
			quilldiv.appendChild(toolbar)
			formstuff.appendChild(textarea)
		</script>
	</body>
	</html>
	`
	return c.HTML(http.StatusOK, fmt.Sprintf(template, style, blurbId, text))
}

func (cs BlurbServer) getRoot(c echo.Context) error {
	c.Response().Header().Set("Cache-control", "no-cache")
	return c.Redirect(301, "/blurb/" + getNewBlurbId())
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


func (cs BlurbServer) getRaw(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := cs.getBlurbText(blurbId)
	return c.String(200, text)
}

func (cs BlurbServer) getBlurb(c echo.Context) error {
	blurbId := c.Param("blurb")
	text := cs.getBlurbText(blurbId)
	template := `
	<head>
		<script type="text/javascript" src="/static/viewfuncs.js"></script>
		<style>
			button { margin: auto; display: block; }
			p { margin: 0em; }
			#text { letter-spacing: 1px; line-height: 1.4; font-family: Sans-serif; word-wrap: break-word; height: auto;}
		</style>
	</head>
	<html>
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<body>
			<div id="all">
				<div id="text">
					%s
				</div>
				<a href="%s"><button>Edit: %s</button></a>
			</div>
		</body>
		<script type="text/javascript">
			fit();
			rawurl = "%s";
			streamurl = "%s"
			watchUpdates(rawurl, streamurl);
		</script>
	</html>
	`
	return c.HTML(http.StatusOK, 
		fmt.Sprintf(template, 
			strings.ReplaceAll(string(text), "\n", "<br>"),
			"/editor/" + blurbId,
			"/blurb/" + blurbId,
			"/raw/" + blurbId,
			"/stream/" + blurbId))
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
		return c.Redirect(http.StatusSeeOther, "/blurb/" + blurbId)
	} else {
		return c.String(400, err.Error())
	}
}
