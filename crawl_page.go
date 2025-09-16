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

type Config struct {
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

func (cfg Config) String() string {
	var ret string
	ret += fmt.Sprintf("\nbaseURL: \"%s\"", cfg.baseURL)
	ret += fmt.Sprintf("\nmaxDepth: %d", cfg.maxDepth)
	ret += fmt.Sprintf("\nmaxPages: %d", cfg.maxPages)
	ret += fmt.Sprintf("\nmaxGoroutines: %d", cap(cfg.concurrencyControl))
	loString := cfg.lo.String()
	loString = strings.ReplaceAll(loString, "\n", "\n  ")
	ret += fmt.Sprintf("\nloggingOptions: {%s\n}", loString)

	return ret
}

func ConfigSetup(
	maxDepth int, maxPages int32, maxGoroutines int,
	baseURL string, logger *log.Logger, logOptions *LoggingOptions,
	storePage func(url, html string, siteLinks, imgLinks []string) error,
	isPageStored func(url string) (bool, error)) (cfg *Config, err error) {

	cfg = &Config{}
	cfg.wg = &sync.WaitGroup{}

	cfg.baseURL, err = url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Base url parse error: %v", err)
	}

	cfg.concurrencyControl = make(chan struct{}, maxGoroutines)
	cfg.maxDepth = maxDepth
	cfg.maxPages = maxPages
	cfg.storePage = storePage
	cfg.isPageStored = isPageStored
	cfg.lo = logOptions
	cfg.log = logger

	return cfg, nil
}

func resolveRelativeURLs() {}

func (cfg *Config) logError(message, rawCurrentURL string, err error) {
	cfg.log.Printf(" - error  : \"%s\" :ERROR: %s%s\n", rawCurrentURL, message, err)
}

func (cfg *Config) CrawlPage(rawCurrentURL string) {
	var currentDepth int
	var pageCount int32

	fmt.Println("tis workin")
	if cfg.lo.DoLogging && cfg.lo.DoStart {
		cfgString := cfg.String()
		cfgString = strings.ReplaceAll(cfgString, "\n", "\n  ")
		cfg.log.Printf("starting crawl of: \"%s\" with config: {%s\n}", cfg.baseURL, cfgString)
		cfg.log.Print("\n----------\n\n")
	}

	cfg.concurrencyControl <- struct{}{}
	cfg.wg.Add(1)
	go cfg.internalCrawlPage(rawCurrentURL, currentDepth, &pageCount)
	cfg.wg.Wait()

	if cfg.lo.DoLogging && cfg.lo.DoEnd {
		cfg.log.Print("\n----------\n\n")
		cfg.log.Println("end of crawl")
		cfg.log.Print("\n----------\n\n")
	}
}

// Todo: add resoleRelativeURLs here
func (cfg *Config) internalCrawlPage(rawCurrentURL string, currentDepth int, pageCount *int32) {
	defer cfg.wg.Done()
	defer func() { <-cfg.concurrencyControl }()
	atomic.AddInt32(pageCount, 1)

	currentDepth += 1

	// enforce same host for base and current URLs, Part 1
	parsedCurrentUrl, err := url.Parse(rawCurrentURL)
	if err != nil {
		if cfg.lo.DoLogging && cfg.lo.DoErrors {
			cfg.logError("current url parse error: ", rawCurrentURL, err)
		}
		return
	}

	// normalise current URL
	currentUrl, err := normalizeURL(rawCurrentURL)
	if err != nil {
		if cfg.lo.DoLogging && cfg.lo.DoErrors {
			cfg.logError("normalise current url error: ", rawCurrentURL, err)
		}
		return
	}

	// check if page is already crawled
	doDepthLog := cfg.lo.DoLogging && cfg.lo.DoDepth
	exists, err := cfg.isPageStored(currentUrl)
	if err != nil {
		cfg.logError("isPageStored callback error: ", currentUrl, err)
	}
	if exists {
		if doDepthLog {
			cfg.log.Printf(" - depth %d: \"%s\"\n", currentDepth, rawCurrentURL)
		}
		return
	} else {
		if doDepthLog {
			cfg.log.Printf(" - DEPTH %d: \"%s\"\n", currentDepth, rawCurrentURL)
		}
	}

	atomInt := atomic.LoadInt32(pageCount)
	if atomInt > cfg.maxPages && atomInt <= 0 {
		if cfg.lo.DoLogging && cfg.lo.DoPageAbyss {
			cfg.log.Printf("   aby-s: \"%s\"\n", rawCurrentURL)
		}
		return
	}

	if cfg.maxDepth > 0 && currentDepth > cfg.maxDepth {
		if cfg.lo.DoLogging && cfg.lo.DoDepthAbyss {
			cfg.log.Printf("   abyss: \"%s\"\n", rawCurrentURL)
		}
		return
	}

	// enforce same host for base and current URLs, Part 2
	if cfg.baseURL.Host != parsedCurrentUrl.Host {
		if cfg.lo.DoLogging && cfg.lo.DoErrors {
			err := fmt.Errorf(
				"host difference: Base: \"%s\", Current: \"%s\"",
				cfg.baseURL.Host, parsedCurrentUrl.Host)
			cfg.logError("", rawCurrentURL, err)
		}
		return
	}

	// parse links of current webpage
	webpage, err := getHTML(rawCurrentURL)
	if err != nil {
		if cfg.lo.DoLogging && cfg.lo.DoErrors {
			cfg.logError("getHTML error: ", rawCurrentURL, err)
		}
		return
	}
	gotURLs, err := getURLsFromHTML(webpage, rawCurrentURL)
	if err != nil {
		if cfg.lo.DoLogging && cfg.lo.DoErrors {
			cfg.logError("getURLsFromHTML error: ", rawCurrentURL, err)
		}
		return
	}

	gotImgLinks := []string{}

	cfg.storePage(currentUrl, webpage, gotURLs, gotImgLinks)

	// go deeper
	func() { <-cfg.concurrencyControl }()
	for _, v := range gotURLs {
		cfg.concurrencyControl <- struct{}{}
		cfg.wg.Add(1)

		if cfg.lo.DoLogging && cfg.lo.DoWidth {
			cfg.log.Printf("   width  : \"%s\"", v)
		}
		go cfg.internalCrawlPage(v, currentDepth, pageCount)
	}
	cfg.concurrencyControl <- struct{}{}
}
