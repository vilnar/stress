# stress-tester
Pluggable benchmarking tool which can do multi processing and measure throughput and latency.
It is intentionally written to benchmark manticore daemon.
As prominent feature it is possibility to cross-bench different protocols manticore supports, and not just public mysql/http, but also ancient sphinx API and even sphinx API internode flavour (the one used by distr index to connect to agents).

## installation

Assume that you have installed go lang on your test platform (visit download section of https://golang.org for it).
Then perform this couple of commands:

```
go get -u github.com/manticoresoftware/stress
```

## usage
```
Usage: stress [-?] [-b N] [-c N] [--csv] [--data path/to/data] [--from N] [-h mysql|plain3|json|fjson|http|fhttp|api] [--limit N] [--tag tag] [parameters ...]
 -?, --help
 -b N           batch size (1 by default)
 -c N           concurrency (1 by default)
     --csv      will output only final result in csv compatible format
     --data=path/to/data
                path to data dir or file
     --from=N   N starts with defined document
 -h, --plugin=mysql|plain3|json|fjson|http|fhttp|api
                name of plugin
     --limit=N  N max number of documents to process
     --tag=tag  add tag to the csv output

After all opts for the app you can also provide options for plugins:
	--host=127.0.0.1	host where daemon listens
	--port		port where daemon listens (default 9306 for mysql, 9308 for http, 9312 for api)
	--index=idx	name of the index to query
	--maxmatches	maxmatches or limit param. 

For mysql plugin you can also provide:
	--filter	clause which will be appended after 'WHERE MATCH()'...

Available plugins:
	mysql	executes queries via sphinxql, may use filters
	plain3	hardcoded mysql to 127.0.0.1:9306, index lj, limit 100000 (no options available)
	json	executes queries via http, /search/json endpoint
	fjson	same as json, but works using fasthttp package
	http	executes queries via http, /search endpoint
	fhttp	same as http, but works using fasthttp package
	api	executes queries via classic binary sphinx API proto as distr works with agents
	apiclient	executes queries via classic binary sphinx API proto, as php and another APIs
```

## Few examples

```
./stress -h api --limit 100000 -b2 -c10 --data /work/stress/ljquerylog.txt.gz --index lj
Time elapsed: 0s, throughput (curr / from start): 0 / 0 rps, 10 children running, 0 elements processed
Time elapsed: 1s, throughput (curr / from start): 2744 / 2744 rps, 10 children running, 2752 elements processed
Time elapsed: 2s, throughput (curr / from start): 2660 / 2702 rps, 10 children running, 5408 elements processed
Time elapsed: 3s, throughput (curr / from start): 3018 / 2807 rps, 10 children running, 8428 elements processed
...
Time elapsed: 38s, throughput (curr / from start): 2607 / 2518 rps, 10 children running, 95712 elements processed
Time elapsed: 39s, throughput (curr / from start): 2505 / 2518 rps, 10 children running, 98220 elements processed
Finally time elapsed: 39.8s, final throughput 2510 rps, 99922 elements processed

FINISHED. Total time: 39.8s throughput: 2510 rps
Worker api
Latency stats:
	count:	99922 latencies analyzed
	avg:	3.969ms
	median:	1.450ms
	95p:	11.044ms
	99p:	45.840ms

Plugin's output:
	Count:	99922
	Total matches:	734164

```

```
snikolaev@dev:~/stress_tester$ ./stress -h api --limit 10000 --csv -b2 -c10 --data /work/stress/ljquerylog.txt.gz --index lj
                               95p latency, ms;99p latency, ms;avg latency, ms;batch size;concurrency;elements count;latencies count;median latency, ms;throughput;total time
                               10.296013;36.874722999999996;3.642010261256754;2;10;9994;9994;1.369458;2666;3.748s
```

```
$ cat query.log
/* Mon Jan 10 13:22:50.297 2021 conn 11 real 0.000 wall 0.000 found 10 */ SELECT * FROM lj;
/* Mon Jan 10 13:22:58.944 2021 conn 11 real 0.000 wall 0.000 found 1 */ SELECT count(*) FROM lj;

$ ./stress -h mysqlplain --limit 100000 -b2 -c10 --data query.log --host 127.0.0.1 --port 9306
```

```
$ cat queries_match.sql
@(producer_title,producer_title) (asus)
@(producer_title,producer_title) (asus rog)

$ ./stress -h mysql --limit 100000 -b2 -c10 --data queries_match.sql --host 127.0.0.1 --port 9306 --index lj
```
