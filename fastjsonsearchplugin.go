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
	"github.com/valyala/fasthttp"
	"os"
	"time"
)

type fastjson struct {
	url, idx, maxmatches string
	client               *fasthttp.Client
}

func getFastjson() *fastjson {
	var S fastjson
	return &S
}

func (this *fastjson) init(opts getopt.Set) {

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

func (this *fastjson) setup(opts interface{}) {
	*this = *opts.(*fastjson)
	this.client = &fasthttp.Client{}
}

func (this *fastjson) query(queries *[]string) []queryInfo {

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

		req := fasthttp.AcquireRequest()
		req.SetRequestURI(this.url)
		req.Header.SetMethod("POST")
		req.Header.SetContentType("application/json")
		req.SetBody([]byte(jbody))

		resp := fasthttp.AcquireResponse()
		err := this.client.Do(req, resp)

		if err != nil {
			fmt.Println("Failed for", query, err)
			continue
		}

		var bodyBytes []byte
		if resp.Header.StatusCode() == fasthttp.StatusOK {
			bodyBytes = resp.Body()
		}

		var dat map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &dat); err != nil {
			fmt.Println("Failed to parse json for", query, err)
			continue
		}

		auto := int64(dat["took"].(float64)) * int64(time.Millisecond)
		results = append(results, queryInfo{latency: time.Duration(auto), numRows: len(dat["hits"].(map[string]interface{})["hits"].([]interface{}))})
	}
	return results
}

func (this *fastjson) close() {
}
