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
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pborman/getopt/v2"
)

type sqlPlainPlug struct {
	dsn      string
	sphinxql *sql.DB
}

func createSqlPlainPlug() *sqlPlainPlug {
	return &sqlPlainPlug{}
}

func (this *sqlPlainPlug) init(opts getopt.Set) {
	sHost := getopt.StringLong("host", 'H', "127.0.0.1")
	iPort := getopt.IntLong("port", 'P', 9306)
	if err := opts.Getopt(os.Args, nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		getopt.Usage()
		os.Exit(1)
	}

	this.dsn = fmt.Sprintf("u:p@tcp(%s:%d)/t", *sHost, *iPort)
}

func (this *sqlPlainPlug) setup(opts interface{}) {
	a := opts.(*sqlPlainPlug)
	this.dsn = a.dsn
	db, err := sql.Open("mysql", this.dsn)
	if err != nil {
		panic(err.Error())
	}

	this.sphinxql = db
}

func (this *sqlPlainPlug) query(queries *[]string) []queryInfo {
	results := make([]queryInfo, 0, len(*queries))
	for _, query := range *queries {
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
	// fmt.Println (results)
	return results
}

func (this *sqlPlainPlug) close() {
	_ = this.sphinxql.Close()
}
