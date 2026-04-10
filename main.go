package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type SafeMap struct {
	mu sync.Mutex
	visited map[string]bool
	host string
}

func (s *SafeMap) ShouldVisit(rawUrl string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", false
	}

	if u.Host != "" && u.Host != s.host {
		return "", false
	}

	u.Fragment = ""
	normalized := strings.TrimSuffix(u.String(), "/")

	if s.visited[normalized] || normalized == "" {
		return "", false
	}

	s.visited[normalized] = true
	return normalized, true
}

// FETCH LINKS THE ACTION FUNCTION
// HITS THE NETWORK AND RETURNS A SLICE OF DISCOVERED STRINGS
func fetchLinks(targetUrl string) []string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(targetUrl)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	return discoverLinks(resp.Body, targetUrl)
}

// DISCOVERLINKS THE PARSER
// STREAM HTML AND EXTRACTS HREF ATTRIBUTES
func discoverLinks(body io.Reader, baseUrl string) []string {
	var links []string
	tokenizer := html.NewTokenizer(body)
	base, _ := url.Parse(baseUrl)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return links
		}

		token := tokenizer.Token()
		if tokenType == html.StartTagToken && token.Data == "a" {
			for _, attr := range token.Attr {
				if attr.Key == "href" {
					val, err := url.Parse(attr.Val)
					if err == nil {
						links = append(links, base.ResolveReference(val).String())
					}
				}
			}
		}
	}
}


func worker(id int, jobs <- chan string, results chan <- []string, wg sync.WaitGroup) {
	for link
}
