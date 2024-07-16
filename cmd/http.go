package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"golang.org/x/net/publicsuffix"
)

func getDefaultHTTPClient() (*http.Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	cookies := []*http.Cookie{
		{Name: "RF_ACCESS_LEVEL", Value: os.Getenv("RF_ACCESS_LEVEL")},
		{Name: "RF_AUTH", Value: os.Getenv("RF_AUTH")},
		{Name: "RF_W_AUTH", Value: os.Getenv("RF_W_AUTH")},
		{Name: "RF_SECURE_AUTH", Value: os.Getenv("RF_SECURE_AUTH")},
		{Name: "RF_PARTY_ID", Value: os.Getenv("RF_PARTY_ID")},
	}
	u, err := url.ParseRequestURI("http://www.redfin.com")
	if err != nil {
		return nil, err
	}
	jar.SetCookies(u, cookies)
	hc := &http.Client{
		Jar: jar,
	}
	return hc, nil
}
