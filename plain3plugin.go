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
	"time"
)

type plain3 struct {
	port     int
	idx      string
	sphinxql *sql.DB
}

func getPlain3() *plain3 {
	var S plain3
	return &S
}

func (this *plain3) init(opts getopt.Set) {
}

func (this *plain3) setup(opts interface{}) {

	this.port = 9306
	this.idx = "lj"
	dsn := fmt.Sprintf("u:p@tcp(127.0.0.1:%d)/t", this.port)
	db, err := sql.Open("mysql", dsn)

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	this.sphinxql = db
}

func (this *plain3) query(queries *[]string) []queryInfo {

	results := make([]queryInfo, 0, len(*queries))
	for _, query := range *queries {

		query := fmt.Sprintf("SELECT * FROM %s WHERE MATCH('%s') limit 100000 option max_matches=100000", this.idx, Escape(query))
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

func (this *plain3) close() {
	_ = this.sphinxql.Close()
}
