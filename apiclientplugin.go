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
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pborman/getopt/v2"
	"os"
	"time"
)

type apiclientopts struct {
	maxmatches int
	uri, idx   string
}

type apiclientplug struct {
	apiclientopts
	isReady bool
	client  SphinxClient
}

func getClientApi() *apiclientplug {

	var S apiclientplug
	return &S
}

func (this *apiclientopts) init(opts getopt.Set) {

	sHost := getopt.StringLong("host", 'H', "127.0.0.1")
	iPort := getopt.IntLong("port", 'P', 9312)
	sIndex := getopt.StringLong("index", 0, "idx")
	iMaxmatches := getopt.IntLong("maxmatches", 0, 0)

	if err := opts.Getopt(os.Args, nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		getopt.Usage()
		os.Exit(1)
	}

	this.uri = fmt.Sprintf("%s:%d", *sHost, *iPort)

	this.idx = *sIndex

	if *iMaxmatches <= 0 {
		this.maxmatches = 20
	} else {
		this.maxmatches = *iMaxmatches
	}
}

func (this *apiclientplug) setup(opts interface{}) {

	a := opts.(*apiclientplug)

	this.apiclientopts = a.apiclientopts
	var err error
	err = this.client.Connect(this.uri)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}

	err = this.client.SendHandshake()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	} else {
		this.isReady = true
	}
}

func (this *apiclientplug) query(queries *[]string) []queryInfo {

	results := make([]queryInfo, 0, len(*queries))
	if !this.isReady {
		return results
	}

	for _, query := range *queries {

		start := time.Now()
		count, msg, err := this.client.SendClientSearch(this.idx, query, this.maxmatches)
		elapsed := time.Since(start)

		if msg != "" {
			fmt.Println("Remote error:", msg)
		}

		if err != nil {
			fmt.Println("Agent error:", err, "for query", query)
		} else {
			results = append(results, queryInfo{latency: elapsed, numRows: count})
		}
	}
	return results
}

func (this *apiclientplug) close() {
	_ = this.client.Close()
}
