package main

import (
	"strings"

	"golang.org/x/net/html"
)

func removeDuplicates(input []string) []string {
	uniqueMap := make(map[string]bool)
	var result []string

	for _, str := range input {
		if !uniqueMap[str] {
			uniqueMap[str] = true
			result = append(result, str)
		}
	}

	return result
}

// use a map instead of string list ?
func extractURLsFromNode(node *html.Node, rawBaseURL string, urls *[]string) {
	rawBaseURL = strings.TrimRight(rawBaseURL, "/")

	if node.Type == html.ElementNode && node.Data == "a" {
		for _, a := range node.Attr {
			if a.Key == "href" {

				put := strings.TrimSpace(a.Val)
				if put == "" {
					break
				}
				if strings.HasPrefix(put, "/") {
					put = rawBaseURL + put
				}

				*urls = append(*urls, put)
				break
			}
		}
	}

	if node.NextSibling != nil {
		extractURLsFromNode(node.NextSibling, rawBaseURL, urls)
	}
	if node.FirstChild != nil {
		extractURLsFromNode(node.FirstChild, rawBaseURL, urls)
	}
}

func getURLsFromHTML(htmlBody, baseURL string) (gotURLs []string, err error) {
	htmlReader := strings.NewReader(htmlBody)
	rootNode, err := html.Parse(htmlReader)

	extractURLsFromNode(rootNode, baseURL, &gotURLs)
	gotURLs = removeDuplicates(gotURLs)

	return gotURLs, err
}
