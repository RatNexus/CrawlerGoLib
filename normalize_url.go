package main

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

func normalizeURL(inputURL string) (outputURL string, err error) {
	var outputURLslice []string = make([]string, 4)

	//to lowercase
	lowURL := strings.ToLower(inputURL)

	urlStruct, err := url.Parse(lowURL)
	if err != nil {
		return "", err
	}

	//remove scheme
	outputURLslice[0] = urlStruct.Hostname()

	//remove default port
	port := urlStruct.Port()
	if port == "443" {
		port = ""
	}
	if port != "" {
		port = ":" + port
	}
	outputURLslice[1] = port

	//remove slash multiples
	urlPath := urlStruct.Path
	re := regexp.MustCompile(`/+`)
	urlPath = re.ReplaceAllString(urlPath, "/")

	//remove dot-segments
	urlSplit := strings.Split(urlPath, "/../")
	urlPath = strings.Join(urlSplit, "/")

	urlSplit = strings.Split(urlPath, "/./")
	urlPath = strings.Join(urlSplit, "/")

	//remove trailing slash
	urlPath = strings.TrimSuffix(urlPath, "/")

	//remove default filename
	urlPath = strings.TrimSuffix(urlPath, "/index.html")
	outputURLslice[2] = urlPath

	//sort query parameters
	queryPar := urlStruct.RawQuery
	values, err := url.ParseQuery(queryPar)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(values))

	for k, v := range values {
		sort.Strings(v)
		values[k] = v
	}

	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryParSlice = make([]string, 0)
	for _, k := range keys {
		for _, v := range values[k] {
			queryParSlice = append(queryParSlice, fmt.Sprintf("%s=%s", k, v))
		}
	}
	if len(queryParSlice) > 0 {
		queryPar = strings.Join(queryParSlice, "&")
		outputURLslice[3] = "?" + queryPar
	}

	outputURL = strings.Join(outputURLslice, "")
	return outputURL, err
}
