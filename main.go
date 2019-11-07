package main

// USAGE:
// protoc --plugin ./protoc-gen-k8s --k8s_out=. --k8s_opt=group=drekle.example.io examples/gcp.proto

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/drekle/protoc-gen-k8s/pkg/generator"
	"github.com/gogo/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	req := &plugin.CodeGeneratorRequest{}
	resp := &plugin.CodeGeneratorResponse{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	err = req.Unmarshal(data)
	if err != nil {
		panic(err)
	}

	//Assert that a group has been set as a parameter
	parameters := req.GetParameter()
	options := make(map[string]string)
	groupkv := strings.Split(parameters, ",")
	for _, element := range groupkv {
		kv := strings.Split(element, "=")
		if len(kv) > 1 {
			options[kv[0]] = kv[1]
		}
	}

	gen, err := generator.NewControllerGenerator(req, resp, options)
	if err != nil {
		panic(err)
	}
	err = gen.GenerateCode()
	if err != nil {
		panic(err)
	}

	marshalled, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(marshalled)
	println()
	println("In the output directory you can now run `make all`.")
}
