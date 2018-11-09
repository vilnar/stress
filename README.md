Test for https://stackoverflow.com/questions/53079379/stack-overflow-parallel-updating-one-rt-index/53170038?noredirect=1#comment93260683_53170038

This is a test to test behavior of high concurrency inserting into Manticore/Sphinx RT index
How to run:
```
git clone https://github.com/Ivinco/stress-tester.git -b test_rt_insert_with_docker
cd stress-tester
docker-compose up --build
```
This will run Manticore/Sphinx search, run test with 10 parallel workers that overall insert 10000 documents each containing 'test' in 2 fields and 1 string attribute.

Then:
```
mysql -P9306 -h0
```
and verify the results.

