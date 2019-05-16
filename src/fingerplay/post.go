package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "code.google.com/p/log4go"
)

func Post(url string, arg interface{}, reply interface{}) (err error) {
	var (
		response *http.Response
		body     []byte
		buf      *bytes.Buffer
	)

	if arg != nil {
		if b, ok := arg.([]byte); !ok {
			if body, err = json.Marshal(arg); err != nil {
				return
			}
		} else {
			body = b
		}
	}

	buf = bytes.NewBuffer(body)

	if response, err = http.Post(url, "application/json; charset=utf-8", buf); err != nil {
		return
	}

	if response != nil {
		defer response.Body.Close()
	} else {
		return
	}

	if body, err = ioutil.ReadAll(response.Body); err != nil {
		return
	}

	log.Debug("post: %s, response data: %s", url, body)
	return json.Unmarshal(body, reply)
}
