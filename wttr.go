package main

import (
	"encoding/json"
	"fmt"
	"github.com/justjanne/powerline-go/powerline"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

const LOCATION = "WTTR_LOCATION"
const FORMAT = "WTTR_FORMAT"
const TIMEOUT = "WTTR_TIMEOUT"
const CACHE = "WTTR_CACHE"

var cacheLocation = path.Join(os.Getenv("HOME"), ".wttr_cache")
var format = "%l:+%c+%f"
var timeout = "5m"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-help" {
		usage(0)
	}

	location := ""

	if loc := os.Getenv(CACHE); loc != "" {
		cacheLocation = loc
	}
	if location = os.Getenv(LOCATION); location == "" {
		usage(1)
	}
	if f := os.Getenv(FORMAT); f != "" {
		format = f
	}
	w, err := getWttr(location, format)
	if err != nil {
		log.Fatal(err)
	}
	out, err := json.Marshal([]WttrCache{w})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(out))
}

type WttrCache struct {
	powerline.Segment
	Location    string
	LastRefresh time.Time
}

func getWttr(location, format string) (WttrCache, error) {
	if w, ok := checkCache(location); ok {
		return w, nil
	}
	get, err := http.Get(fmt.Sprintf("https://wttr.in/%s?format=%s", location, format))
	if err != nil {
		return WttrCache{}, err
	}
	resp, err := io.ReadAll(get.Body)
	if err != nil {
		return WttrCache{}, err
	}
	cache := WttrCache{
		Segment: powerline.Segment{
			Name:    "wttr",
			Content: string(resp),
		},
		Location:    location,
		LastRefresh: time.Now(),
	}
	w, err := json.Marshal(cache)
	if err != nil {
		return WttrCache{}, err
	}
	err = os.WriteFile(cacheLocation, w, 0666)
	if err != nil {
		return WttrCache{}, err
	}
	return cache, nil
}

func checkCache(location string) (WttrCache, bool) {
	cache := WttrCache{}
	res, err := os.ReadFile(cacheLocation)
	if err != nil {
		log.Printf("Error reading cache file: %v", err)
		return WttrCache{}, false
	}
	err = json.Unmarshal(res, &cache)
	if err != nil {
		log.Printf("Error parsing .wttr_cache file: %s", err)
		return WttrCache{}, false
	}
	if cache.Location != location {
		return WttrCache{}, false
	}
	if t := os.Getenv(TIMEOUT); t != "" {
		timeout = t
	}
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		log.Printf("Error parsing duration: %s", err)
		return WttrCache{}, false
	}
	if cache.LastRefresh.Add(duration).Before(time.Now()) {
		return WttrCache{}, false
	}
	return cache, true
}

func usage(exitCode int) {
	fmt.Println("powerline-go-wttr usage:")
	fmt.Println("  wttr -help")
	fmt.Println("Environment Variables:")
	fmt.Printf("  %s: required airport, zip, or city name\n", LOCATION)
	fmt.Printf("  %s: optional wttr.in format string (default: %s)\n", FORMAT, format)
	fmt.Printf("  %s: go duration for cache invalidation (default: %s)\n", TIMEOUT, timeout)
	fmt.Printf("  %s: location for cache (default: %s)\n", CACHE, cacheLocation)
	os.Exit(exitCode)
}
