package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/osmike/fcache"
)

func fetchDataFromRemote(timeMS int) (string, error) {
	url := fmt.Sprintf("https://www.bluebricks.co/golang?time=%d", timeMS)

	client := &http.Client{
		Timeout: time.Duration(timeMS+5000) * time.Millisecond,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func main() {
	timeMS := 2000

	cachedFn := fcache.NewCachedFunction(fetchDataFromRemote, nil, nil)
	fmt.Printf("[%v] Starting first request to remote service with time %d ms...\n", time.Now().Truncate(time.Second), timeMS)
	data, err := cachedFn(timeMS)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Printf("[%v] First request completed, received data: %s\n", time.Now().Truncate(time.Second), data)
	fmt.Printf("[%v] Starting second request to remote service with time %d ms...\n", time.Now().Truncate(time.Second), timeMS)
	data, err = cachedFn(timeMS)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Printf("[%v] Second request completed, received data: %s\n", time.Now().Truncate(time.Second), data)

	fmt.Printf("Received data: %s\n", data)
}
