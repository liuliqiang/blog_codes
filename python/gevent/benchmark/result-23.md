gevent WSGIServer

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


Server Software:
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        31 bytes

Concurrency Level:      100
Time taken for tests:   7.575 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      158000 bytes
HTML transferred:       31000 bytes
Requests per second:    132.02 [#/sec] (mean)
Time per request:       757.464 [ms] (mean)
Time per request:       7.575 [ms] (mean, across all concurrent requests)
Transfer rate:          20.37 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.8      0       5
Processing:    14  718 133.4    743     848
Waiting:       13  718 133.4    743     847
Total:         18  718 132.6    743     848

Percentage of the requests served within a certain time (ms)
  50%    743
  66%    756
  75%    766
  80%    775
  90%    798
  95%    824
  98%    836
  99%    841
 100%    848 (longest request)
