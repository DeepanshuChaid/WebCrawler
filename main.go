package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func (s *SafeMap) ShouldVisit(rawUrl, currentHost string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// SO THE URL PARSE CREATES A URL STRUCT IN THAT PARSE WE CUT THE # PART OF THE URL AS WELL IN THE U WE GET THE URL STRUCT ITSELF
	u, err := url.Parse(rawUrl)
	// WHAT DOES HOST MEANS host" or "host:port" (see Hostname and Port methods) <- THIS IS WHATS WRITTEN IN THE DOCS
	if err != nil || (u.Host != "" && u.Host != currentHost) {
		return "", false
	}

	// NORMALIZED: HANDLE RELATIVE LINKS (/PKG -> HTTPS://GO.DEV/PKG)
	// AND REMOVE TRAILING SLASHES SO /DOC ETC ARE REMOVED FROM THE URL PREVENTING DUPLICATION
	normalized := strings.TrimSuffix(rawUrl, "/")

	if s.visited[normalized] {
		return "", false
	}

	s.visited[normalized] = true
	return normalized, true
}

func worker(id int, jobs <-chan string, results chan <- []string, wg *sync.WaitGroup) {
	for link := range jobs {
		fmt.Println("Worker ", id, " is hitting: ", link)

		foundLinks := fetchLinks(link)
		results <- foundLinks
		wg.Done()
	}
}

func fetchLinks(l string) string  {
	return "nil"
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
	// Instead of loading the entire HTML file into memory at once, the tokenizer "streams" through it, breaking the text into manageable pieces called tokens (like start tags, end tags, or text).
	tokenizer := html.NewTokenizer(body)

	for {
		// HERE WE GOT A SINGLE TOKEN
		tokenType := tokenizer.Next()
		// WE CHECK IF THE TOKEN TYPE EOF WHICH MEANS NO BITCHES🤨 (NO MORE TOKEN LEFT)
		if tokenType == html.ErrorToken {
			return links // End of document
		}

		// HERE WE GET THE TYPE OF TOKEN EG:- P, H, DIV TAGS
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
	startURL := "https://go.dev" // Start with Go docs

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
