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
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/pborman/getopt/v2"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type queryInfo struct {
	latency time.Duration
	numRows int
}

type queryInfos []queryInfo

func (q queryInfos) Len() int {
	return len(q)
}

func (q queryInfos) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q queryInfos) Less(i, j int) bool {
	return q[i].latency < q[j].latency
}

func report(results *[]queryInfo) map[string]int {
	totalmatches := 0
	for _, query := range *results {
		totalmatches += query.numRows
	}
	return map[string]int{"Total matches": totalmatches, "Count": len(*results)}
}

type stressplugin interface {

	// parse provided command line parameters (m.b. specific for filter) and once save the values
	// it is run in 1 thread, 1 time, no other things like starting connections, etc. need to be here
	init(opts getopt.Set)

	// start all necessary things like db connection. view param pass parsed option from init step
	// that method called from goroutine, so is multi-threaded
	setup(view interface{})

	// called from worker. Queries contain batch (>=1) string queries. So, method must return appropriate
	// number of results in queryInfo
	query(queries *[]string) (results []queryInfo)

	// called at the end of bench. Close the connection, release resources, etc.
	close()

	// kinda 'static'. Whole resultset (of all threads) passed inside, all necessary final calculations to be done there.
	//report(results *[]queryInfo) map[string]int
}

func makeplug(name string) (stressplugin, error) {
	var res stressplugin
	res = nil
	switch name {
	case "mysql":
		res = getSql()
	case "plain3":
		res = getPlain3()
	case "json":
		res = getEs()
	case "fjson":
		res = getFastjson()
	case "http":
		res = getHttpSearch()
	case "fhttp":
		res = getFastHttpSearch()
	case "api":
		res = getApi()
	case "apiclient":
		res = getClientApi()
	}
	if res == nil {
		return nil, errors.New(fmt.Sprintln("Can't create plugin", name, "available values are mysql, plain3, json, fjson, http, fhttp, api, apiclient"))
	}
	return res, nil
}

func feed_text(txt io.Reader, ifrom, ilimit int, feed chan<- string) (err error) {
	r := bufio.NewReader(txt)
	curs, printed := 0, 0
	s, e := r.ReadString('\n')
	for e == nil {
		curs += 1
		if ilimit != -1 && printed >= ilimit {
			return io.EOF
		}

		if ifrom == -1 || ifrom < curs {
			str := strings.TrimRight(s, "\n")
			if str != "" {
				feed <- str
				printed++
			} else {
				curs--
			}
		}
		s, e = r.ReadString('\n')
	}
	return e
}

func feed_csv(txt io.Reader, ifrom, ilimit int, feed chan<- string) (err error) {
	r := csv.NewReader(txt)
	curs, printed := 0, 0
	s, e := r.Read()
	for e == nil {
		for _, ones := range s {
			curs++
			if ilimit != -1 && printed >= ilimit {
				return io.EOF
			}

			if ifrom == -1 || ifrom < curs {
				if ones != "" {
					feed <- ones
					printed++
				} else {
					curs--
				}
			}
		}
		s, e = r.Read()
	}
	return e
}

func feed_file(path string, ifrom, ilimit int, feed chan<- string) error {

	var result io.Reader
	bottomfile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer bottomfile.Close()

	result = bottomfile

	if filepath.Ext(path) == ".gz" {
		gr, err := gzip.NewReader(bottomfile)
		if err != nil {
			return err
		}
		defer gr.Close()

		result = gr
		path = path[:len(path)-3]
	}

	if filepath.Ext(path) == ".csv" {
		return feed_csv(result, ifrom, ilimit, feed)
	}
	return feed_text(result, ifrom, ilimit, feed)
}

func feed_dir(path string, ifrom, ilimit int, feed chan<- string) (err error) {

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	if ifrom > 0 {
		files = files[ifrom:]
	}

	if ilimit > 0 {
		files = files[:ilimit]
	}

	for _, f := range files {
		fname := filepath.Join(path, f.Name())
		err = feed_file(fname, -1, -1, feed)
		if err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

func filtershelp() {
	fmt.Println("\nAfter all opts for the app you can also provide options for plugins:")
	fmt.Println("\t--host=127.0.0.1\thost where daemon listens")
	fmt.Println("\t--port	\tport where daemon listens (default 9306 for mysql, 9308 for http, 9312 for api)")
	fmt.Println("\t--index=idx\tname of the index to query")
	fmt.Println("\t--maxmatches\tmaxmatches or limit param. ")
	fmt.Println("\nFor mysql plugin you can also provide:")
	fmt.Println("\t--filter\tclause which will be appended after 'WHERE MATCH()'...")
	fmt.Println("\nAvailable plugins:")
	fmt.Println("\tmysql\texecutes queries via sphinxql, may use filters")
	fmt.Println("\tplain3\thardcoded mysql to 127.0.0.1:9306, index lj, limit 100000 (no options available)")
	fmt.Println("\tjson\texecutes queries via http, /search/json endpoint")
	fmt.Println("\tfjson\tsame as json, but works using fasthttp package")
	fmt.Println("\thttp\texecutes queries via http, /search endpoint")
	fmt.Println("\tfhttp\tsame as http, but works using fasthttp package")
	fmt.Println("\tapi\texecutes queries via classic binary sphinx API proto as distr works with agents")
	fmt.Println("\tapiclient\texecutes queries via classic binary sphinx API proto, as php and another APIs")
}

func main() {

	bHelp := getopt.BoolLong("help", '?', "", "display help")
	sPlugin := getopt.StringLong("plugin", 'h', "", "name of plugin", "mysql|plain3|json|fjson|http|fhttp|api")
	sData := getopt.StringLong("data", 0, "", "path to data dir or file", "path/to/data")
	iLimit := getopt.IntLong("limit", 0, -1, "N max number of documents to process", "N")
	iFrom := getopt.IntLong("from", 0, -1, "N starts with defined document", "N")
	iBatch := getopt.Int('b', 1, "batch size (1 by default)", "N")
	bCsv := getopt.BoolLong("csv", 0, "will output only final result in csv compatible format")
	iConcurrency := getopt.Int('c', 1, "concurrency (1 by default)", "N")
	sTag := getopt.StringLong("tag", 0, "", "add tag to the csv output", "tag")

	var opts = getopt.CommandLine
	_ = opts.Getopt(os.Args, nil)

	if *bHelp {
		getopt.Usage()
		filtershelp()

		os.Exit(0)
	}
	if *bHelp || *sPlugin == "" || *sData == "" {
		getopt.Usage()
		filtershelp()
		if *bHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	fi, err := os.Lstat(*sData)
	if err != nil {
		log.Fatal(err)
	}

	src := make(chan string, *iBatch)

	// this routine reads all the sources and feeds the src channel
	go func() {
		defer close(src)
		var err error
		if fi.Mode().IsDir() {
			err = feed_dir(*sData, *iFrom, *iLimit, src)
		} else if fi.Mode().IsRegular() {
			err = feed_file(*sData, *iFrom, *iLimit, src)
		}

		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
	}()

	template, err := makeplug(*sPlugin)
	if err != nil {
		panic(err)
	}

	template.init(*opts)

	resultchan := make(chan interface{}, *iConcurrency*10)
	var vg sync.WaitGroup
	vg.Add(*iConcurrency)

	for i := 0; i < *iConcurrency; i++ {
		go func() { // this routine(s) do the benching itself, taking query strings from src channel
			executor, err := makeplug(*sPlugin)
			if err != nil {
				panic(err)
			}
			executor.setup(template)
			buf := make([]string, *iBatch)
			i := 0
			for x := range src {
				if i >= *iBatch {
					i = 0
					resultchan <- executor.query(&buf)
				}
				buf[i] = x
				i += 1
			}
			if i != 0 {
				buf := buf[:i]
				resultchan <- executor.query(&buf)
			}
			executor.close()
			vg.Done()
		}()
	}

	var collector []queryInfo
	ticker := time.NewTicker(1 * time.Second)
	startTime := time.Now()
	checkTime := startTime
	prevlatency, abslatency := 0, 0

	// this routine collect all results from siblings workers, and also type statistic every second to console
	go func() {
		if !*bCsv {
			fmt.Printf("Time elapsed: %s, throughput (curr / from start): 0 / 0 rps, %d children running, 0 elements processed\n", time.Since(startTime).Round(time.Second), *iConcurrency)
		}

		for results := range resultchan {
			select {
			case <-ticker.C:
				if !*bCsv {
					elapsed := time.Since(startTime).Round(time.Second)
					curvalue := math.Floor(float64(prevlatency) / time.Since(checkTime).Seconds())
					absvalue := math.Floor(float64(abslatency) / time.Since(startTime).Seconds())
					fmt.Printf("Time elapsed: %s, throughput (curr / from start): %.0f / %.0f rps, %d children running, %d elements processed\n", elapsed, curvalue, absvalue, *iConcurrency, abslatency)
					prevlatency = 0
					checkTime = time.Now()
				}
			default:
				for _, result := range results.([]queryInfo) {
					collector = append(collector, result)
					prevlatency++
					abslatency++
				}
			}
		}
	}()

	vg.Wait()
	close(resultchan)

	collected := len(collector)
	fcollected := float32(collected)

	if collected == 0 {
		return
	}

	// first, type out final  statistic
	rawelapsed := time.Since(startTime)
	elapsed := rawelapsed.Round(time.Millisecond)
	absvalue := math.Floor(float64(abslatency) / time.Since(startTime).Seconds())
	if !*bCsv {
		fmt.Printf("Finally time elapsed: %s, final throughput %.0f rps, %d elements processed\n", elapsed, absvalue, abslatency)
	}

	sort.Sort(queryInfos(collector))
	fromplug := report(&collector)

	var totallatency time.Duration
	for _, q := range collector {
		totallatency += q.latency
	}

	result := make(map[string]interface{})
	median := int(0.5 * fcollected)
	p95 := int(0.95 * fcollected)
	p99 := int(0.99 * fcollected)

	result["concurrency"] = *iConcurrency
	result["batch size"] = *iBatch
	result["total time"] = elapsed.Round(time.Millisecond)
	result["throughput"] = absvalue
	result["elements count"] = collected
	result["latencies count"] = collected
	result["avg latency, ms"] = 1000.0 * totallatency.Seconds() / float64(collected)
	result["median latency, ms"] = collector[median].latency.Seconds() * 1000
	result["95p latency, ms"] = collector[p95].latency.Seconds() * 1000
	result["99p latency, ms"] = collector[p99].latency.Seconds() * 1000

	if *bCsv {
		if *sTag != "" {
			result["tag"] = *sTag
		}
		var keys []string
		for key := range result {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Println(strings.Join(keys, ";"))

		var values []string
		for _, key := range keys {
			values = append(values, fmt.Sprint(result[key]))
		}
		fmt.Println(strings.Join(values, ";"))
	} else {
		fmt.Println("\nFINISHED. Total time:", elapsed, "throughput:", absvalue, "rps")
		fmt.Println("Worker", *sPlugin)
		fmt.Println("Latency stats:")
		fmt.Printf("\tcount:\t%d latencies analyzed\n", collected)
		fmt.Printf("\tavg:\t%.3fms\n", result["avg latency, ms"])
		fmt.Printf("\tmedian:\t%.3fms\n", result["median latency, ms"])
		fmt.Printf("\t95p:\t%.3fms\n", result["95p latency, ms"])
		fmt.Printf("\t99p:\t%.3fms\n", result["99p latency, ms"])
		fmt.Println("\nPlugin's output:")
		for key, value := range fromplug {
			fmt.Printf("\t%s:\t%d\n", key, value)
		}
	}
}
