package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func getHTML(rawURL string) (webpage string, err error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// wait... will this `` string work?
		return "", fmt.Errorf(`got error status code: "%d"`, resp.StatusCode)
	}

	contentTypeString := resp.Header.Get("Content-Type")
	contentTypes := strings.Split(contentTypeString, ";")
	hasTextType := false
	for _, t := range contentTypes {
		t = strings.Trim(t, " ")
		if t == "text/html" {
			hasTextType = true
			break
		}
	}
	if !hasTextType {
		return "", fmt.Errorf("content type header is not text/html is \"%s\"", contentTypeString)
	}

	webpageBytes, errReadAll := io.ReadAll(resp.Body)
	if errReadAll != nil {
		return "", fmt.Errorf("read all error: %e", errReadAll)
	}
	webpage = string(webpageBytes)
	return webpage, err
}
