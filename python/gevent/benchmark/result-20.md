nginx + gunicorn

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


Server Software:        nginx/1.8.0
Server Hostname:        127.0.0.1
Server Port:            8080

Document Path:          /
Document Length:        31 bytes

Concurrency Level:      100
Time taken for tests:   3.078 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      179000 bytes
HTML transferred:       31000 bytes
Requests per second:    324.88 [#/sec] (mean)
Time per request:       307.802 [ms] (mean)
Time per request:       3.078 [ms] (mean, across all concurrent requests)
Transfer rate:          56.79 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   0.8      0       4
Processing:    42  291  43.1    289     410
Waiting:       35  291  43.1    289     410
Total:         46  291  42.7    290     411
WARNING: The median and mean for the initial connection time are not within a normal deviation
        These results are probably not that reliable.

Percentage of the requests served within a certain time (ms)
  50%    290
  66%    312
  75%    325
  80%    331
  90%    338
  95%    344
  98%    357
  99%    381
 100%    411 (longest request)
