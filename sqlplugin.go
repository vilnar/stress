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
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pborman/getopt/v2"
	"os"
	"time"
)

type sqlopts struct {
	maxmatches, filter, idx, dsn string
}

type sqlplug struct {
	sqlopts
	sphinxql *sql.DB
}

func Escape(sql string) string {
	dest := make([]byte, 0, 2*len(sql))
	var escape byte
	for i := 0; i < len(sql); i++ {
		c := sql[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
			break
		case '\n': /* Must be escaped for logs */
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '"': /* Better safe than sorry */
			escape = '"'
			break
		case '\032': /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			dest = append(dest, '\\', escape)
		} else {
			dest = append(dest, c)
		}
	}

	return string(dest)
}

func getSql() *sqlplug {
	var S sqlplug
	return &S
}

func (this *sqlopts) init(opts getopt.Set) {

	sHost := getopt.StringLong("host", 'H', "127.0.0.1")
	iPort := getopt.IntLong("port", 'P', 9306)
	sIndex := getopt.StringLong("index", 0, "idx")
	iMaxmatches := getopt.IntLong("maxmatches", 0, -1)
	sFilter := getopt.StringLong("filter", 0, "")

	if err := opts.Getopt(os.Args, nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		getopt.Usage()
		os.Exit(1)
	}

	this.filter = ""
	if *sFilter != "" {
		this.filter = " AND " + *sFilter
	}
	this.idx = *sIndex

	this.maxmatches = ""
	if *iMaxmatches >= 0 {
		this.maxmatches = fmt.Sprintf(" limit %d option max_matches=%d", *iMaxmatches, *iMaxmatches)
	}

	this.dsn = fmt.Sprintf("u:p@tcp(%s:%d)/t", *sHost, *iPort)
}

func (this *sqlplug) setup(opts interface{}) {

	a := opts.(*sqlplug)

	this.sqlopts = a.sqlopts
	db, err := sql.Open("mysql", this.dsn)

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	this.sphinxql = db
}

func (this *sqlplug) query(queries *[]string) []queryInfo {

	results := make([]queryInfo, 0, len(*queries))
	for _, query := range *queries {

		query := fmt.Sprintf("SELECT * FROM %s WHERE MATCH('%s')%s%s", this.idx, Escape(query), this.filter, this.maxmatches)
		start := time.Now()
		rows, _ := this.sphinxql.Query(query)
		elapsed := time.Since(start)

		count := 0
		if rows != nil {
			for rows.Next() {
				count++
			}
			_ = rows.Close()
		}

		results = append(results, queryInfo{latency: elapsed, numRows: count})
	}
	//	fmt.Println (results)
	return results
}

func (this *sqlplug) close() {
	_ = this.sphinxql.Close()
}
