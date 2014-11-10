package main

import (
	"net/url"
)


// Returns the value of the first id= parameter of the URL.
func getChannelId(u *url.URL) string {
	// FIXME: Configurable id parameter!
	return getURLParameter(u,"id")
}

// Returns the value of the parameter *key* from URL *u*
func getURLParameter(u *url.URL, key string) string {
	return getURLParameters(u,key)[0]
}

// Return all values with the key *key*
func getURLParameters(u *url.URL, key string) []string {
	v, err := url.ParseQuery(u.RawQuery)

	if err != nil {
		return []string{}
	}

	return v[key]
}

// If all else is not enough: Just get the parameter map
// (intended to be used if more than one parameter is needed
// so the URL doesn't need to be parsed more than once)
func getAllURLParameters(u *url.URL) map[string][]string {
	v, err := url.ParseQuery(u.RawQuery)

	if err != nil {
		return nil
	}
	return v
}

