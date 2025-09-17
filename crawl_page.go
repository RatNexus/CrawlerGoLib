package crawler

//TODO: Error setting exit on callback
//TODO: Crawler does not honor given limits. Please Fix
//TODO:	Implement Robots.txt support
//TODO: Implement and Use resolveRelativeURLs
//TODO: Implement imgLinksGet for var gotImgLinks
//TODO: maxPageCount should return early when at limit

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

type LoggingOptions struct {
	DoLogging    bool `json:"doLogging"`
	DoStart      bool `json:"doStart"`
	DoEnd        bool `json:"doEnd"`
	DoSummary    bool `json:"doSummary"`
	DoPageAbyss  bool `json:"doPageAbyss"`
	DoDepthAbyss bool `json:"doDepthAbyss"`
	DoDepth      bool `json:"doDepth"`
	DoWidth      bool `json:"doWidth"`
	DoErrors     bool `json:"doErrors"`
	DoPages      bool `json:"doPages"`
	DoIdRoutine  bool `json:"doIdRoutine"`
}

func (lo LoggingOptions) String() string {
	var ret string
	ret += fmt.Sprintf("\nDoLogging: %t", lo.DoLogging)
	ret += fmt.Sprintf("\nDoStart: %t", lo.DoStart)
	ret += fmt.Sprintf("\nDoEnd: %t", lo.DoEnd)
	ret += fmt.Sprintf("\nDoSummary: %t", lo.DoSummary)
	ret += fmt.Sprintf("\nDoPageAbyss: %t", lo.DoPageAbyss)
	ret += fmt.Sprintf("\nDoDepthAbyss: %t", lo.DoDepthAbyss)
	ret += fmt.Sprintf("\nDoDepth: %t", lo.DoDepth)
	ret += fmt.Sprintf("\nDoWidth: %t", lo.DoWidth)
	ret += fmt.Sprintf("\nDoErrors: %t", lo.DoErrors)
	ret += fmt.Sprintf("\nDoPages: %t", lo.DoPages)
	ret += fmt.Sprintf("\nDoIdRoutine: %t", lo.DoIdRoutine)

	return ret
}

type Crawler struct {
	baseURL            *url.URL
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
	maxPages           int32
	maxDepth           int
	lf                 *os.File
	log                *log.Logger
	lo                 *LoggingOptions
	storePage          func(url, html string, siteLinks, imgLinks []string) error
	isPageStored       func(url string) (bool, error)
}

type Config struct {
	BaseURL        string          `json:"baseUrl"`
	MaxDepth       int             `json:"maxDepth"`
	MaxPages       int             `json:"maxPages"`
	MaxGoroutines  int             `json:"maxGoroutines"`
	LoggingOptions *LoggingOptions `json:"loggingOptions"`
	Logger         *log.Logger
}

func (crw Crawler) String() string {
	var ret string
	ret += fmt.Sprintf("\nbaseURL: \"%s\"", crw.baseURL)
	ret += fmt.Sprintf("\nmaxDepth: %d", crw.maxDepth)
	ret += fmt.Sprintf("\nmaxPages: %d", crw.maxPages)
	ret += fmt.Sprintf("\nmaxGoroutines: %d", cap(crw.concurrencyControl))
	loString := crw.lo.String()
	loString = strings.ReplaceAll(loString, "\n", "\n  ")
	ret += fmt.Sprintf("\nloggingOptions: {%s\n}", loString)

	return ret
}

func (cfg *Config) MakeCrawler(
	storePage func(url, html string, siteLinks, imgLinks []string) error,
	isPageStored func(url string) (bool, error)) (crw *Crawler, err error) {

	crw = &Crawler{}
	crw.wg = &sync.WaitGroup{}

	crw.baseURL, err = url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("Base url parse error: %v", err)
	}

	crw.concurrencyControl = make(chan struct{}, cfg.MaxGoroutines)
	crw.maxDepth = cfg.MaxDepth
	crw.maxPages = int32(cfg.MaxPages)
	crw.storePage = storePage
	crw.isPageStored = isPageStored
	crw.lo = cfg.LoggingOptions
	crw.log = cfg.Logger

	return crw, nil
}

func resolveRelativeURLs() {}

func (crw *Crawler) logError(message, rawCurrentURL string, err error) {
	crw.log.Printf(" - error  : \"%s\" :ERROR: %s%s\n", rawCurrentURL, message, err)
}

func (crw *Crawler) CrawlPage(rawCurrentURL string) {
	var currentDepth int
	var pageCount int32

	fmt.Println("tis workin")
	if crw.lo.DoLogging && crw.lo.DoStart {
		cfgString := crw.String()
		cfgString = strings.ReplaceAll(cfgString, "\n", "\n  ")
		crw.log.Printf("starting crawl of: \"%s\" with config: {%s\n}", crw.baseURL, cfgString)
		crw.log.Print("\n----------\n\n")
	}

	crw.concurrencyControl <- struct{}{}
	crw.wg.Add(1)
	go crw.internalCrawlPage(rawCurrentURL, currentDepth, &pageCount)
	crw.wg.Wait()

	if crw.lo.DoLogging && crw.lo.DoEnd {
		crw.log.Print("\n----------\n\n")
		crw.log.Println("end of crawl")
		crw.log.Print("\n----------\n\n")
	}
}

// Todo: add resoleRelativeURLs here
func (crw *Crawler) internalCrawlPage(rawCurrentURL string, currentDepth int, pageCount *int32) {
	defer crw.wg.Done()
	defer func() { <-crw.concurrencyControl }()
	atomic.AddInt32(pageCount, 1)

	currentDepth += 1

	// enforce same host for base and current URLs, Part 1
	parsedCurrentUrl, err := url.Parse(rawCurrentURL)
	if err != nil {
		if crw.lo.DoLogging && crw.lo.DoErrors {
			crw.logError("current url parse error: ", rawCurrentURL, err)
		}
		return
	}

	// normalise current URL
	currentUrl, err := normalizeURL(rawCurrentURL)
	if err != nil {
		if crw.lo.DoLogging && crw.lo.DoErrors {
			crw.logError("normalise current url error: ", rawCurrentURL, err)
		}
		return
	}

	// check if page is already crawled
	doDepthLog := crw.lo.DoLogging && crw.lo.DoDepth
	exists, err := crw.isPageStored(currentUrl)
	if err != nil {
		crw.logError("isPageStored callback error: ", currentUrl, err)
	}
	if exists {
		if doDepthLog {
			crw.log.Printf(" - depth %d: \"%s\"\n", currentDepth, rawCurrentURL)
		}
		return
	} else {
		if doDepthLog {
			crw.log.Printf(" - DEPTH %d: \"%s\"\n", currentDepth, rawCurrentURL)
		}
	}

	atomInt := atomic.LoadInt32(pageCount)
	if atomInt > crw.maxPages && atomInt <= 0 {
		if crw.lo.DoLogging && crw.lo.DoPageAbyss {
			crw.log.Printf("   aby-s: \"%s\"\n", rawCurrentURL)
		}
		return
	}

	if crw.maxDepth > 0 && currentDepth > crw.maxDepth {
		if crw.lo.DoLogging && crw.lo.DoDepthAbyss {
			crw.log.Printf("   abyss: \"%s\"\n", rawCurrentURL)
		}
		return
	}

	// enforce same host for base and current URLs, Part 2
	if crw.baseURL.Host != parsedCurrentUrl.Host {
		if crw.lo.DoLogging && crw.lo.DoErrors {
			err := fmt.Errorf(
				"host difference: Base: \"%s\", Current: \"%s\"",
				crw.baseURL.Host, parsedCurrentUrl.Host)
			crw.logError("", rawCurrentURL, err)
		}
		return
	}

	// parse links of current webpage
	webpage, err := getHTML(rawCurrentURL)
	if err != nil {
		if crw.lo.DoLogging && crw.lo.DoErrors {
			crw.logError("getHTML error: ", rawCurrentURL, err)
		}
		return
	}
	gotURLs, err := getURLsFromHTML(webpage, rawCurrentURL)
	if err != nil {
		if crw.lo.DoLogging && crw.lo.DoErrors {
			crw.logError("getURLsFromHTML error: ", rawCurrentURL, err)
		}
		return
	}

	gotImgLinks := []string{}

	crw.storePage(currentUrl, webpage, gotURLs, gotImgLinks)

	// go deeper
	func() { <-crw.concurrencyControl }()
	for _, v := range gotURLs {
		crw.concurrencyControl <- struct{}{}
		crw.wg.Add(1)

		if crw.lo.DoLogging && crw.lo.DoWidth {
			crw.log.Printf("   width  : \"%s\"", v)
		}
		go crw.internalCrawlPage(v, currentDepth, pageCount)
	}
	crw.concurrencyControl <- struct{}{}
}
