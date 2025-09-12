package main

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
	"sync"
	"sync/atomic"
)

type LoggingOptions struct {
	DoLogging    bool
	DoStart      bool
	DoEnd        bool
	DoSummary    bool
	DoPageAbyss  bool
	DoDepthAbyss bool
	DoDepth      bool
	DoWidth      bool
	DoErrors     bool
	DoPages      bool
	DoIdRoutine  bool
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
		// make this message variable depending on other logging options
		cfg.log.Printf("starting crawl of: \"%s\" with max depth of %d\n",
			cfg.baseURL, cfg.maxDepth)
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
