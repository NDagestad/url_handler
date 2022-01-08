package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type URL struct {
	raw_url string
	*url.URL
}

func Parse(raw_url string) (*URL, error) {
	var new_url string
	parsed, err := url.Parse(raw_url)
	if err != nil {
		// TODO do something if the parsing fails
		log("Got an error while parsning: %v\n", err)
	}

	if parsed.Scheme == "" {
		_, err := os.Open(raw_url)
		if !errors.Is(err, os.ErrNotExist) { // If it exists, it is a file
			log("Expanding potential tildes in the path\n")
			raw_url = expandTilde(raw_url)
			raw_url, err = filepath.Abs(raw_url)
			log("Adding the file:// scheme to the url for disambiguation\n")
			new_url = fmt.Sprintf("file://%s", raw_url)
		} else { // If not we assume it is a URL
			log("Adding the http:// scheme to the url for disambiguation\n")
			new_url = fmt.Sprintf("http://%s", raw_url)
		}
		parsed, err = url.Parse(new_url)
		if err != nil {
			// TODO Trickle down this error message instead of directly printing it?
			log("Error in reparsing the url after expanding the tilde %v\n", err)
		}
	}

	return &URL{
		raw_url: new_url,
		URL:     parsed,
	}, err
}
