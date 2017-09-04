gunicorn multithread(4 worker 12 thread)


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
Time taken for tests:   2.182 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      183000 bytes
HTML transferred:       31000 bytes
Requests per second:    458.27 [#/sec] (mean)
Time per request:       218.213 [ms] (mean)
Time per request:       2.182 [ms] (mean, across all concurrent requests)
Transfer rate:          81.90 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0   17  20.7      5      81
Processing:    14  193  97.0    178     578
Waiting:       14  177  95.7    160     576
Total:         16  210  92.5    197     583

Percentage of the requests served within a certain time (ms)
  50%    197
  66%    235
  75%    260
  80%    279
  90%    338
  95%    380
  98%    440
  99%    483
 100%    583 (longest request)
