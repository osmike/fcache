package main

import (
	"fmt"
	"time"

	"github.com/osmike/fcache"
)

func main() {
	cachedFunction := fcache.NewCachedFunction(heavyComputation, nil, nil)
	fmt.Printf("[%v] Starting heavy computation...\n", time.Now().Truncate(time.Second))
	res, err := cachedFunction(2000 * time.Millisecond)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("[%v] Heavy computation completed, result - %s.\n", time.Now().Truncate(time.Second), res)

	fmt.Printf("[%v] Starting cached heavy computation...\n", time.Now().Truncate(time.Second))

	_, err = cachedFunction(2000 * time.Millisecond)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("[%v] Heavy computation completed, result cached - %s.\n", time.Now().Truncate(time.Second), res)
}

func heavyComputation(t time.Duration) (string, error) {
	time.Sleep(t)
	return "cached value", nil
}
