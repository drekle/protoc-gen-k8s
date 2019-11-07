package template

var MAKE_TEMPLATE = `.PHONY: all
all: init generate build

.PHONY: init
init:
	chmod a+x hack/update-codegen.sh
	mkdir -p vendor/k8s.io
	if [ ! -d vendor/k8s.io/code-generator ]; then git clone https://github.com/kubernetes/code-generator.git vendor/k8s.io/code-generator; fi

.PHONY: generate
generate:
	cd hack; ./update-codegen.sh

.PHONY: build
build: 
	cd cmd; GOOS=linux go build .
`
