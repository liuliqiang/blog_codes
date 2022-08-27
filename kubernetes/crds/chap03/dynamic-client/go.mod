module github.com/liuliqiang/blog-demos/kubernetes/crds/chap03

go 1.13

require (
	github.com/liuliqiang/log4go v0.0.0-20191118103554-a6fc3169999a
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/client-go v11.0.0+incompatible
)

replace (
	github.com/kubernetes/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
)
