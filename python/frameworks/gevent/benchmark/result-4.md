run with following codes with gunicorn webserver 

``
gunicorn -w 4 -b localhost:5000 app:app
```


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
Document Length:        23 bytes

Concurrency Level:      100
Time taken for tests:   530.287 seconds
Complete requests:      1000
Failed requests:        0
Total transferred:      175000 bytes
HTML transferred:       23000 bytes
Requests per second:    1.89 [#/sec] (mean)
Time per request:       53028.711 [ms] (mean)
Time per request:       530.287 [ms] (mean, across all concurrent requests)
Transfer rate:          0.32 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   1.5      0      11
Processing:  1646 49941 12553.7  51099   66564
Waiting:     1645 49940 12553.7  51099   66564
Total:       1651 49941 12552.7  51099   66564

Percentage of the requests served within a certain time (ms)
  50%  51099
  66%  56936
  75%  59210
  80%  60004
  90%  62900
  95%  64404
  98%  65229
  99%  65572
 100%  66564 (longest request)
