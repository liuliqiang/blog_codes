gunicorn gevent(4 worker + gevent)

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
Time taken for tests:   2.565 seconds
Complete requests:      1000
Failed requests:        50
   (Connect: 0, Receive: 0, Length: 50, Exceptions: 0)
Non-2xx responses:      50
Total transferred:      196650 bytes
HTML transferred:       44000 bytes
Requests per second:    389.85 [#/sec] (mean)
Time per request:       256.509 [ms] (mean)
Time per request:       2.565 [ms] (mean, across all concurrent requests)
Transfer rate:          74.87 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   1.1      0       4
Processing:     9  248 171.1    196    1801
Waiting:        8  247 171.1    195    1801
Total:          9  248 171.2    196    1801

Percentage of the requests served within a certain time (ms)
  50%    196
  66%    249
  75%    323
  80%    360
  90%    499
  95%    570
  98%    675
  99%    757
 100%   1801 (longest request)
