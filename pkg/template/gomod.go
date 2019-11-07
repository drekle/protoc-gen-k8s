package template

var GOMOD_TEMPLATE = `module {{ .RepoURL }}

go 1.12

require (
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/grpc-gateway v1.11.3
	github.com/imdario/mergo v0.3.7 // indirect
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/genproto v0.0.0-20190927181202-20e1ac93f88c
	google.golang.org/grpc v1.24.0
	k8s.io/api v0.0.0-20190718183219-b59d8169aab5 // indirect
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/code-generator v0.0.0-20190927075303-016f2b3d74d0
	k8s.io/utils v0.0.0-20190923111123-69764acb6e8e // indirect
)
`
