package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mmcdole/gofeed"
)

func retrieveSMTPPassword(serverURI string, sessionKey string) (password string, error error) {

	uri := fmt.Sprintf("%s/%s/%s", serverURI, "services/storage/passwords", "html_email_smtp")

	res, err := sendRequest(sessionKey, "GET", uri, nil)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	fp := gofeed.NewParser()
	feed, _ := fp.ParseString((string(data)))

	for _, item := range feed.Items {

		var extension Extension
		xml.Unmarshal([]byte(item.Content), &extension)

		keys := extension.Keys

		for _, key := range keys {
			switch key.Name {
			case "clear_password":
				return key.Value, nil
			}
		}
	}

	return "", nil
}

func sendRequest(sessionKey string, method string, uri string, body io.Reader) (*http.Response, error) {

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Splunk "+sessionKey)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil

}

type Extension struct {
	Keys []Key `xml:"key"`
}

type Key struct {
	Value string `xml:",chardata"`
	Name  string `xml:"name,attr"`
	List  List   `xml:"list"`
	Dict  Dict   `xml:"dict"`
}

type List struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Value string `xml:",chardata"`
	Name  string `xml:"name,attr"`
}

type Dict struct {
	Keys []Key `xml:"key"`
}
