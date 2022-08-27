module github.com/liuliqiang/blog-demos/kubernetes/crds/chap03/code-gen

go 1.13

require (
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/liuliqiang/log4go v0.0.0-20191118103554-a6fc3169999a
	golang.org/x/oauth2 v0.0.0-20191122200657-5d9234df094c // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	k8s.io/api v0.0.0-20191121015604-11707872ac1c // indirect
	k8s.io/apimachinery v0.0.0-20191123233150-4c4803ed55e3
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 // indirect
)

replace (
	github.com/kubernetes/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
)
