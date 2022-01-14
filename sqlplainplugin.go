//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License. You should have
// received a copy of the GPL license along with this program; if you
// did not, you can find it at http://www.gnu.org/
//

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pborman/getopt/v2"
)

type sqlPlainPlug struct {
	dsn      string
	timeout  time.Duration
	tryLimit int
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
	this.timeout = 1500 * time.Millisecond
	this.tryLimit = 3
}

func (this *sqlPlainPlug) setup(opts interface{}) {
	a := opts.(*sqlPlainPlug)
	this.dsn = a.dsn
	this.timeout = a.timeout
	this.tryLimit = a.tryLimit
	for i := 1; i <= this.tryLimit; i++ {
		db, err := sql.Open("mysql", this.dsn)
		if err != nil {
			log.Println("Error: not create connect to db, try", i)
			continue
		}

		ctx, _ := context.WithTimeout(context.Background(), this.timeout)
		err = db.PingContext(ctx)
		if err != nil {
			log.Println("Error: not ping to db, try", i)
			db.Close()
			continue
		}

		this.sphinxql = db
		return
	}
	panic("not connected")
}

func (this *sqlPlainPlug) query(queries *[]string) []queryInfo {
	results := make([]queryInfo, 0, len(*queries))
	for _, query := range *queries {
		func() {
			start := time.Now()
			rows, err := this.sphinxql.Query(query)
			if err != nil {
				// log.Println(err)
				return
			}
			defer rows.Close()
			elapsed := time.Since(start)

			count := 0
			for rows.Next() {
				count++
			}
			if err = rows.Err(); err != nil {
				log.Println(err)
				return
			}

			results = append(results, queryInfo{latency: elapsed, numRows: count})
		}()
	}
	return results
}

func (this *sqlPlainPlug) close() {
	_ = this.sphinxql.Close()
}
