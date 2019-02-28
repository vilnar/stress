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
	"os"
	"strings"
	"time"
)

type esplug struct {
	url, idx, maxmatches string
	client               *http.Client
}

func getEs() *esplug {
	var S esplug
	return &S
}

func (this *esplug) init(opts getopt.Set) {

	sHost := getopt.StringLong("host", 'H', "127.0.0.1")
	iPort := getopt.IntLong("port", 'P', 9308)
	sIndex := getopt.StringLong("index", 0, "idx")
	iMaxmatches := getopt.IntLong("maxmatches", 0, 0)

	if err := opts.Getopt(os.Args, nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		getopt.Usage()
		os.Exit(1)
	}

	this.maxmatches = ""
	if *iMaxmatches != 0 {
		this.maxmatches = fmt.Sprintf("%d", *iMaxmatches)
	}

	this.idx = *sIndex
	this.url = fmt.Sprintf("http://%s:%d/json/search", *sHost, *iPort)
}

func (this *esplug) setup(opts interface{}) {
	*this = *opts.(*esplug)
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1024,
		DisableKeepAlives:   false,
	}
	this.client = &http.Client{Transport: tr}
}

func (this *esplug) query(queries *[]string) []queryInfo {

	results := make([]queryInfo, 0, len(*queries))
	for _, query := range *queries {

		var jbody string
		jquery, _ := json.Marshal(query)
		sjquery := string(jquery)
		if this.maxmatches != "" {
			jbody = `{"index":"` + this.idx + `","query":{"match":{"_all":{"query":` + sjquery + `,"operator":"and"}},"limit":` + this.maxmatches + `}`
		} else {
			jbody = `{"index":"` + this.idx + `","query":{"match":{"_all":{"query":` + sjquery + `,"operator":"and"}}}}`
		}

		req, _ := http.NewRequest("POST", this.url, strings.NewReader(jbody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := this.client.Do(req)
		if err != nil {
			fmt.Println("Failed for", query, err)
			continue
		}

		var bodyBytes []byte
		if resp.StatusCode == http.StatusOK {
			bodyBytes, _ = ioutil.ReadAll(resp.Body)
		}
		_ = resp.Body.Close()

		var dat map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &dat); err != nil {
			fmt.Println("Failed to parse json for", query, jbody, bodyBytes, err)
			continue
		}

		auto := int64(dat["took"].(float64)) * int64(time.Millisecond)
		results = append(results, queryInfo{latency: time.Duration(auto), numRows: len(dat["hits"].(map[string]interface{})["hits"].([]interface{}))})
	}
	return results
}

func (this *esplug) close() {
}
