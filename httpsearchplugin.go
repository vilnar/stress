//
// Copyright (c) 2019, Manticore Software LTD (http://manticoresearch.com)
// All rights reserved
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License. You should have
// received a copy of the GPL license along with this program; if you
// did not, you can find it at http://www.gnu.org/
//

package main

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/getopt/v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type httpsearchplug struct {
	url, idx   string
	maxmatches int
	client     *http.Client
}

func getHttpSearch() *httpsearchplug {
	var S httpsearchplug
	return &S
}

func (this *httpsearchplug) init(opts getopt.Set) {

	sHost := getopt.StringLong("host", 'H', "127.0.0.1")
	iPort := getopt.IntLong("port", 'P', 9308)
	sIndex := getopt.StringLong("index", 0, "idx")
	iMaxmatches := getopt.IntLong("maxmatches", 0, 0)

	if err := opts.Getopt(os.Args, nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		getopt.Usage()
		os.Exit(1)
	}

	this.maxmatches = 0
	if *iMaxmatches != 0 {
		this.maxmatches = *iMaxmatches
	}

	this.idx = *sIndex
	this.url = fmt.Sprintf("http://%s:%d/json/search", *sHost, *iPort)
}

func (this *httpsearchplug) setup(opts interface{}) {
	*this = *opts.(*httpsearchplug)
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1024,
		DisableKeepAlives:   false,
	}
	this.client = &http.Client{Transport: tr}
}

func (this *httpsearchplug) query(queries *[]string) []queryInfo {

	results := make([]queryInfo, 0, len(*queries))
	for _, query := range *queries {

		escquery := url.QueryEscape(query)
		var sbody string
		if this.maxmatches != 0 {
			//			sbody = fmt.Sprintf("\"index\"=%s&match=%s&limit=%d&select=*", this.idx, escquery, this.maxmatches)
			sbody = fmt.Sprintf("{\"index\":\"%s\",\"query\":{\"query_string\":\"%s\"},\"limit\":%d}", this.idx, escquery, this.maxmatches)

		} else {
			//			sbody = fmt.Sprintf("index=%s&match=%s&select=*", this.idx, escquery)
			sbody = fmt.Sprintf("{\"index\":\"%s\",\"query\":{\"query_string\":\"%s\"}}", this.idx, escquery)
		}

		req, _ := http.NewRequest("POST", this.url, strings.NewReader(sbody))
		resp, err := this.client.Do(req)
		if err != nil {
			fmt.Println("Failed for", query, err)
			continue
		}

		var bodyBytes []byte
		if resp.StatusCode == http.StatusOK {
			bodyBytes, _ = ioutil.ReadAll(resp.Body)
		} else {
			//			fmt.Println("Error", resp.StatusCode, "for query", query )
			_ = resp.Body.Close()
			continue
		}
		_ = resp.Body.Close()

		var dat map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &dat); err != nil {
			fmt.Println("Failed to parse json for", query, bodyBytes, err)
			continue
		}
		auto := int64(dat["took"].(float64)) * int64(time.Millisecond)
		results = append(results, queryInfo{latency: time.Duration(auto), numRows: len(dat["hits"].(map[string]interface{})["hits"].([]interface{}))})
	}
	return results
}

func (this *httpsearchplug) close() {
}
