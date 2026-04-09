package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// SafeMap prevents race conditions when checking if we've visited a link
type SafeMap struct {
	mu      sync.Mutex
	visited map[string]bool
}

func (s *SafeMap) Visit(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.visited[url] {
		return true
	}
	s.visited[url] = true
	return false
}

func crawl(url string, wg *sync.WaitGroup, jobs chan<- string, tracker *SafeMap) {
	defer wg.Done()

	// 1. Skip if already visited
	if tracker.Visit(url) {
		return
	}

	// 2. Fetch the page
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	fmt.Printf("[✓] Crawled: %s\n", url)
	jobs <- url

	// 3. Parse HTML and find new links
	// (Simplified logic: in a real app, you'd extract text for Cogito here)
	links := discoverLinks(resp.Body)
	for _, link := range links {
		wg.Add(1)
		go crawl(link, wg, jobs, tracker) // Recursive concurrency
	}
}

func discoverLinks(body io.Reader) []string {
	var links []string
	tokenizer := html.NewTokenizer(body)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return links // End of document
		}

		token := tokenizer.Token()
		if tokenType == html.StartTagToken && token.Data == "a" {
			for _, attr := range token.Attr {
				if attr.Key == "href" {
					// In a real app, you'd handle relative URLs here
					links = append(links, attr.Val)
				}
			}
		}
	}
}

func main() {
	startURL := "https://deepanshuchaid.vercel.app" // Start with Go docs

	jobs := make(chan string)
	tracker := &SafeMap{visited: make(map[string]bool)}
	var wg sync.WaitGroup

	wg.Add(1)
	go crawl(startURL, &wg, jobs, tracker)

	// Close channel when all work is done
	go func() {
		wg.Wait()
		close(jobs)
	}()

	// Just a simple counter to see progress
	count := 0
	for range jobs {
		count++
	}
	fmt.Printf("Finished. Total pages found: %d\n", count)
}
