# stress-tester
Pluggable benchmarking tool which can do multi processing and measure throughput and latency

```
Usage: test.php [-h] --plugin=path/to/plugin --data=path/to/data [--limit=N] [--from=N] [-b=N] [-c=N] [--csv]
-b batch size (1 by default)
-c concurrency (1 by default)
--limit max number of documents to process
--from starts with defined document
--csv will output only final result in csv compatible format
```

Few examples:
```
snikolaev@dev:~/stress_tester_github$ ./test.php --plugin=pq_twitter.php --data=/home/snikolaev/twitter/text -b=200 -c=2 --limit=100000
Time elapsed: 0 sec, throughput (curr / from start): 0 / 0 rps, 0 children running, 100000 elements left
Time elapsed: 1.001 sec, throughput (curr / from start): 12187 / 12186 rps, 2 children running, 87400 elements left
Time elapsed: 2.002 sec, throughput (curr / from start): 12387 / 12286 rps, 2 children running, 75000 elements left
Time elapsed: 3.003 sec, throughput (curr / from start): 13190 / 12587 rps, 2 children running, 61800 elements left
Time elapsed: 4.004 sec, throughput (curr / from start): 12992 / 12688 rps, 2 children running, 48800 elements left
Time elapsed: 5.005 sec, throughput (curr / from start): 13180 / 12786 rps, 2 children running, 35600 elements left
Time elapsed: 6.006 sec, throughput (curr / from start): 13190 / 12854 rps, 2 children running, 22400 elements left
Time elapsed: 7.007 sec, throughput (curr / from start): 12591 / 12816 rps, 2 children running, 9800 elements left
Time elapsed: 8.04 sec, throughput (curr / from start): 9868 / 12437 rps, 2 children running, 0 elements left

FINISHED. Total time: 8.094 sec, throughput: 12355 rps
Latency stats:
	count: 100000 latencies analyzed
	avg: 28.617 ms
	median: 28.398 ms
	95p: 37.505 ms
	99p: 42.878 ms

Plugin's output:
	Total matches: 518874
	Count: 100000
```

```
snikolaev@dev:~/stress_tester$ for batchSize in 1 4 5 6 10 20 50 100 200; do ./test.php --plugin=es_pq_twitter.php --data=/home/snikolaev/twitter/text -b=$batchSize -c=8 --limit=10000 --csv; done;
8;1;12.87;777;10000;10000;8.771;4.864;50.212;63.838
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;4;7.98;1253;10000;10000;24.071;12.5;77.675;103.735
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;5;7.133;1401;10000;10000;27.538;15.42;79.169;99.058
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;6;7.04;1420;10000;10000;32.978;19.097;87.458;111.311
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;10;7.374;1356;10000;10000;57.576;51.933;117.053;172.985
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;20;8.642;1157;10000;10000;136.103;125.133;228.399;288.927
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;50;11.565;864;10000;10000;454.78;448.788;659.542;781.465
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;100;25.57;391;10000;10000;1976.077;1110.372;6744.786;7822.412
concurrency;batch size;total time;throughput;elements count;latencies count;avg latency, ms;median latency, ms;95p latency, ms;99p latency, ms
8;200;52.251;191;10000;10000;7957.451;8980.085;9773.551;10167.927
```
