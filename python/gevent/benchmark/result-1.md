run with following codes without any other third tools:

```
app.run(debug=False, threaded=True, processes=1)
```


This is ApacheBench, Version 2.3 <$Revision: 1757674 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking 127.0.0.1 (be patient).....done


Server Software:        Werkzeug/0.12.2
Server Hostname:        127.0.0.1
Server Port:            5000

Document Path:          /
Document Length:        23 bytes

Concurrency Level:      10
Time taken for tests:   37.677 seconds
Complete requests:      100
Failed requests:        0
Total transferred:      16900 bytes
HTML transferred:       2300 bytes
Requests per second:    2.65 [#/sec] (mean)
Time per request:       3767.716 [ms] (mean)
Time per request:       376.772 [ms] (mean, across all concurrent requests)
Transfer rate:          0.44 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:  2661 3725 349.3   3706    4487
Waiting:     2660 3725 349.2   3706    4487
Total:       2661 3726 349.2   3706    4487

Percentage of the requests served within a certain time (ms)
  50%   3706
  66%   3895
  75%   3959
  80%   4057
  90%   4173
  95%   4310
  98%   4391
  99%   4487
 100%   4487 (longest request)
