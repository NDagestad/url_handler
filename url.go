package main

import (
	"errors"
	"path/filepath"
	"net/url"
	"os"
	"fmt"
)

type URL struct {
	raw_url string
	*url.URL
}

func Parse(raw_url string) (*URL, error) {
	parsed, err := url.Parse(raw_url)
	if err != nil {
		// TODO do something if the parsing fails
		log("Got an error while parsning: %v\n", err)
	}


	// TODO Add http/https if the URI is a url without scheme
	if parsed.Scheme == "" {
		_, err := os.Open(raw_url)
		if ! errors.Is(err, os.ErrNotExist) {
			log("Expanding potential tildes in the path\n")
			raw_url = expandTilde(raw_url)
			raw_url, err = filepath.Abs(raw_url)
			log("Adding the file:// scheme to the url for disambiguation\n")
			raw_url = fmt.Sprintf("file://%s", raw_url)
			parsed, err = url.Parse(raw_url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error in reparsing the url after expanding the tilde %v\n", err)
				return &URL{
					raw_url: raw_url,
					URL: parsed,
				}, err
			}
		}
	}

	return &URL{
		raw_url: raw_url,
		URL: parsed,
	}, err
}

//func (this *URL) String() string { return "" }
