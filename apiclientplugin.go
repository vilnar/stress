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
	"github.com/manticoresoftware/go-sdk/manticore"
	"github.com/pborman/getopt/v2"
	"os"
	"time"
)

type apiclientopts struct {
	shost      string
	sport      uint16
	maxmatches int
	idx        string
}

type apiclientplug struct {
	apiclientopts
	isReady bool
	manticore.Client
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

	this.shost = *sHost
	this.sport = uint16(*iPort)

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
	this.SetServer(this.shost, this.sport)
	_, err = this.Open()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}
	this.isReady = true
}

func (this *apiclientplug) query(queries *[]string) []queryInfo {

	results := make([]queryInfo, 0, len(*queries))
	if !this.isReady {
		return results
	}

	start := time.Now()
	searches := make([]manticore.Search, len(*queries))

	for i := 0; i < len(*queries); i++ {

		searches[i] = manticore.NewSearch((*queries)[i], this.idx, "")
		searches[i].MaxMatches = int32(this.maxmatches)
	}

	res, err := this.RunQueries(searches)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Println("Remote error:", err.Error())
	}

	for i := 0; i < len(res); i++ {
		if res[i].Error != "" {
			fmt.Println("Agent error:", res[i].Error, "for query", (*queries)[i])
		} else {
			results = append(results, queryInfo{latency: elapsed, numRows: len(res[i].Matches)})
		}
	}
	return results
}

func (this *apiclientplug) close() {
	_, _ = this.Close()
}
