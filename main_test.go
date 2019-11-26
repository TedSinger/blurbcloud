package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getFreePort() int {
	return 21212
}

func getFileName() string {
	f, _ := ioutil.TempFile("/dev/shm", "*.db")
	return f.Name()
}

func testPostGet(t *testing.T, rootUrl string) {
	expected := "this is some <b>TEXT!</b>"
	_, err := http.Post(rootUrl+"/blurb/foo?text="+url.QueryEscape(expected), "", nil)
	if err != nil {
		panic(err)
	}
	resp, err := http.Get(rootUrl + "/raw/foo")
	if err != nil {
		panic(err)
	}
	actual, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, expected, string(actual))
}

func Test(t *testing.T) {
	port := getFreePort()
	fn := getFileName()
	go run(port, fn)
	rootUrl := fmt.Sprintf("http://localhost:%d", port)
	var err error
	for i := 0; i < 10; i++ {
		_, err := http.Get(rootUrl)
		if err == nil {
			break
		}
		time.Sleep(123 * time.Millisecond)
	}
	if err != nil {
		panic(err)
	}
	testPostGet(t, rootUrl)

}
