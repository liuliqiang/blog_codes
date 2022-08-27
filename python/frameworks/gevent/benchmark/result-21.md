gunicorn

This is ApacheBench, Version 2.3 <$Revision: 1757674 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient)
Completed 100 requests
Completed 200 requests
Completed 300 requests
Completed 400 requests
Completed 500 requests
Completed 600 requests
Completed 700 requests
Completed 800 requests
Completed 900 requests
Completed 1000 requests
Finished 1000 requests


Server Software:        gunicorn/19.7.1
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        31 bytes

Concurrency Level:      100
Time taken for tests:   3.365 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      183000 bytes
HTML transferred:       31000 bytes
Requests per second:    297.18 [#/sec] (mean)
Time per request:       336.493 [ms] (mean)
Time per request:       3.365 [ms] (mean, across all concurrent requests)
Transfer rate:          53.11 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   1.5      0       7
Processing:    18  324 172.1    264     917
Waiting:       18  323 172.1    264     917
Total:         24  324 171.8    264     917

Percentage of the requests served within a certain time (ms)
  50%    264
  66%    272
  75%    287
  80%    294
  90%    609
  95%    784
  98%    870
  99%    905
 100%    917 (longest request)
