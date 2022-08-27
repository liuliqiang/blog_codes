# log4go

[![Go Report Card](https://goreportcard.com/badge/github.com/liuliqiang/log4go)](https://goreportcard.com/report/github.com/liuliqiang/log4go)

A log library for golang based on hashicorp/logutils

## Why log4go

Yes, it another logging library for Go program language. Why do I create a new logging library for Go when there are so many popular logging library. such as:

- [hashicorp/logutils](https://github.com/hashicorp/logutils)
- [sirupsen/logrus](https://github.com/Sirupsen/logrus)
- [golang/glog](https://github.com/golang/glog)
- [so on...](https://github.com/avelino/awesome-go#logging)

For my daily use, i found several points are important for logging:

- level for logging shoud be easy to use
- content for logging should be friendly for human reading
- logs should be easy to check

so i create log4go

## Quick start

1. Step 01: get the library

    ```
    $ go get -u github.com/liuliqiang/log4go
    ```

2. Step 02: try in code:

    ```
    func main() {
    	log4go.Debug("I am debug log")
    	log4go.Info("Web server is started at %s:%d", "127.0.0.1", 80)
    	log4go.Warn("Get an empty http request")
    	log4go.Error("Failed to query record from db: %v", errors.New("db error"))
    }
    ```

3. Step 03: Run it!

    ```
    $ go run main.go
    2019/08/10 00:02:18 [INFO]Web server is started at 127.0.0.1:80
    2019/08/10 00:02:18 [WARN]Get an empty http request
    2019/08/10 00:02:18 [EROR]Failed to query record from db: db error
    ```

## Learn more...

- To be continue...
- More examples at [Examples](./examples)

