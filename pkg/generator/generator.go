package generator

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"path"
	"strings"

	gotemplate "text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	gogen "github.com/golang/protobuf/protoc-gen-go/generator"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"github.com/drekle/protoc-gen-k8s/pkg/template"
)

var EXAMPLE_REPO = "www.github.com/drekle/k8sexample"

type controllerGenerator struct {
	Request  *plugin.CodeGeneratorRequest
	Response *plugin.CodeGeneratorResponse
	Opts     map[string]string
}

const (
	GROUP_OPTION    = "group"
	INTERNAL_FORMAT = "XXX_%s"
)

type LocationMessage struct {
	Location *descriptor.SourceCodeInfo_Location
	Message  *descriptor.DescriptorProto
	Comments []string
}

func validateOptions(opts map[string]string) error {
	for k, _ := range opts {
		found := false
		for _, knownOption := range []string{GROUP_OPTION} {
			if k == knownOption {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Unknown Option `%s`", k)
		}
	}
	return nil
}

func NewControllerGenerator(request *plugin.CodeGeneratorRequest, response *plugin.CodeGeneratorResponse, opts map[string]string) (*controllerGenerator, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}
	// This generator will need to know the output directory
	return &controllerGenerator{
		request,
		response,
		opts,
	}, nil
}

func (c *controllerGenerator) GenerateCode() error {
	// Generate a Kubernetes controller for each protobuf type
	files := make([]*plugin.CodeGeneratorResponse_File, 0)
	c.Response.File = files

	{
		err := c.generateController()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateCobra()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateSignals()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateKubeAPI()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateGoGen()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateGoMod()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateHack()
		if err != nil {
			return err
		}
	}
	{
		err := c.generateMakefile()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controllerGenerator) generateGoGen() error {

	{
		for index, genFile := range c.Request.FileToGenerate {
			proto := c.Request.ProtoFile[index]
			group := c.Opts[GROUP_OPTION]
			group = strings.Replace(group, ".", "", -1)

			newReq := plugin.CodeGeneratorRequest(*c.Request)
			// We must remove all leading comments as to not forward runtime object comments to the kubernetes generator
			for index, _ := range newReq.FileToGenerate {
				proto := newReq.ProtoFile[index]
				desc := proto.GetSourceCodeInfo()
				locations := desc.GetLocation()
				for _, location := range locations {
					comments := strings.Split(location.GetLeadingComments(), "\n")
					for _, comment := range comments {
						if strings.Contains(comment, "k8s.io/apimachinery/pkg/runtime.Object") {
							message := proto.GetMessageType()[location.GetPath()[1]]
							newName := fmt.Sprintf(INTERNAL_FORMAT, message.GetName())
							message.Name = &newName
							proto.GetMessageType()[location.GetPath()[1]] = message
						}
					}
					location.LeadingComments = nil
				}
			}

			newReq.FileToGenerate = []string{genFile}
			// Run the standard gogen to generate the internal types
			g := gogen.New()
			g.Request = &newReq
			g.WrapTypes()
			g.SetPackageNames()
			g.BuildTypeNameMap()
			g.GenerateAllFiles()
			for _, f := range g.Response.File {
				//Override the output file
				newPath := path.Join("pkg", "apis", group, proto.GetPackage(), path.Base(f.GetName()))
				println(newPath)

				c.Response.File = append(c.Response.File, &plugin.CodeGeneratorResponse_File{
					Name:    &newPath,
					Content: f.Content,
				})
			}
		}
	}
	return nil
}

func (c *controllerGenerator) generateHack() error {

	group := c.Opts[GROUP_OPTION]
	tpl := &template.ProtoMessage{
		Group:   strings.Replace(group, ".", "", -1),
		RepoURL: EXAMPLE_REPO,
		Package: "v1",
	}
	{
		hack, err := gotemplate.New("k8s-hack").Funcs(template.FuncMap).Parse(template.K8S_HACK_TEMPLATE)
		if err != nil {
			return err
		}
		filename := "hack/update-codegen.sh"
		err = c.runTemplate(filename, hack, &tpl)
		if err != nil {
			return err
		}
	}
	{
		boilerplate, err := gotemplate.New("k8s-hack").Funcs(template.FuncMap).Parse(template.BOILERPLATE_TEMPLATE)
		if err != nil {
			return err
		}
		filename := "hack/boilerplate.go.txt"
		err = c.runTemplate(filename, boilerplate, &tpl)
		if err != nil {
			return err
		}
	}
	{
		boilerplate, err := gotemplate.New("k8s-hack").Funcs(template.FuncMap).Parse(template.TOOLS_TEMPLATE)
		if err != nil {
			return err
		}
		filename := "hack/tools.go"
		err = c.runTemplate(filename, boilerplate, &tpl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controllerGenerator) generateGoMod() error {

	var tpl template.TemplateOpts
	tpl.RepoURL = EXAMPLE_REPO
	gomod, err := gotemplate.New("GoMod").Funcs(template.FuncMap).Parse(template.GOMOD_TEMPLATE)
	if err != nil {
		return err
	}
	filename := "go.mod"
	err = c.runTemplate(filename, gomod, &tpl)
	if err != nil {
		return err
	}
	return nil
}

func (c *controllerGenerator) generateController() error {

	locationMessageMap := c.getLocationMessage()
	group := c.Opts[GROUP_OPTION]

	for index, filename := range c.Request.FileToGenerate {
		proto := c.Request.ProtoFile[index]
		locationMessages := locationMessageMap[filename]

		var k8stypes template.ProtoFile
		k8stypes.Package = proto.GetPackage()
		k8stypes.Messages = make([]*template.ProtoMessage, 0)
		k8stypes.Group = strings.Replace(group, ".", "", -1)
		k8stypes.RepoURL = EXAMPLE_REPO
		for _, locationMessage := range locationMessages {
			var tpl template.TemplateOpts
			tpl.Name = locationMessage.Message.GetName()
			tpl.Package = proto.GetPackage()
			tpl.RepoURL = EXAMPLE_REPO
			tpl.Group = strings.Replace(group, ".", "", -1)
			tpl.RuntimeType = locationMessage.Message.GetName()

			k8stpl, err := gotemplate.New("K8s-Controller").Funcs(template.FuncMap).Parse(template.ControllerTemplate)
			if err != nil {
				return err
			}
			filename := fmt.Sprintf("pkg/controller/%sController.go", tpl.Name)
			err = c.runTemplate(filename, k8stpl, &tpl)
			if err != nil {
				return err
			}

			entrytpl, err := gotemplate.New("K8s-Entrypoint").Funcs(template.FuncMap).Parse(template.ControllerEntrypoint)
			if err != nil {
				return err
			}
			filename = fmt.Sprintf("pkg/controller/%sEntrypoint.go", tpl.Name)
			err = c.runTemplate(filename, entrytpl, &tpl)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *controllerGenerator) generateMakefile() error {
	{

		filename := fmt.Sprintf("Makefile")
		signals, err := gotemplate.New("Make").Parse(template.MAKE_TEMPLATE)
		if err != nil {
			return err
		}
		var empty struct{}
		err = c.runTemplate(filename, signals, empty)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controllerGenerator) generateSignals() error {
	{

		filename := fmt.Sprintf("pkg/signals/signal.go")
		signals, err := gotemplate.New("Signals").Parse(template.Signals)
		if err != nil {
			return err
		}
		var empty struct{}
		err = c.runTemplate(filename, signals, empty)
		if err != nil {
			return err
		}
	}
	{
		filename := fmt.Sprintf("pkg/signals/signal_posix.go")
		signals, err := gotemplate.New("Signals").Parse(template.SignalsPosix)
		if err != nil {
			return err
		}
		var empty struct{}
		err = c.runTemplate(filename, signals, empty)
		if err != nil {
			return err
		}
	}
	{
		filename := fmt.Sprintf("pkg/signals/signal_windows.go")
		signals, err := gotemplate.New("Signals").Parse(template.SignalsWindows)
		if err != nil {
			return err
		}
		var empty struct{}
		err = c.runTemplate(filename, signals, empty)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controllerGenerator) generateKubeAPI() error {

	locationMessageMap := c.getLocationMessage()
	group := c.Opts[GROUP_OPTION]
	{
		filename := fmt.Sprintf("pkg/apis/%s/register.go", strings.Replace(group, ".", "", -1))
		registerGroup, err := gotemplate.New("K8sGroup").Funcs(template.FuncMap).Parse(template.REGISTER_GROUP_TEMPLATE)
		if err != nil {
			return err
		}
		err = c.runTemplate(filename, registerGroup, template.ProtoMessage{
			Group: group,
		})
		if err != nil {
			return err
		}
	}
	generatedDocPackage := make(map[string]bool)
	for index, filename := range c.Request.FileToGenerate {
		proto := c.Request.ProtoFile[index]
		{
			if _, ok := generatedDocPackage[proto.GetPackage()]; !ok {
				filename := fmt.Sprintf("pkg/apis/%s/%s/doc.go", strings.Replace(group, ".", "", -1), proto.GetPackage())
				registerGroup, err := gotemplate.New("K8sGroup").Funcs(template.FuncMap).Parse(template.DOC_TEMPLATE)
				if err != nil {
					return err
				}
				err = c.runTemplate(filename, registerGroup, template.ProtoMessage{
					Package: proto.GetPackage(),
					Group:   group,
				})
				if err != nil {
					return err
				}
				generatedDocPackage[proto.GetPackage()] = true
			}
		}
		locationMessages := locationMessageMap[filename]

		var k8stypes template.ProtoFile
		k8stypes.Package = proto.GetPackage()
		k8stypes.Messages = make([]*template.ProtoMessage, 0)
		k8stypes.Group = strings.Replace(group, ".", "", -1)
		k8stypes.RepoURL = EXAMPLE_REPO
		for _, locationMessage := range locationMessages {
			message := &template.ProtoMessage{}
			message.Name = locationMessage.Message.GetName()
			message.RuntimeType = fmt.Sprintf(INTERNAL_FORMAT, locationMessage.Message.GetName())
			message.LeadingComments = locationMessage.Comments
			k8stypes.Messages = append(k8stypes.Messages, message)
		}
		filename := fmt.Sprintf("pkg/apis/%s/%s/%sTypes.go", strings.Replace(group, ".", "", -1), proto.GetPackage(), strings.Replace(path.Base(filename), ".proto", "", -1))
		types, err := gotemplate.New("Types").Funcs(template.FuncMap).Parse(template.K8S_TYPE_TEMPLATE)
		if err != nil {
			return err
		}
		err = c.runTemplate(filename, types, k8stypes)
		if err != nil {
			return err
		}
		//Generate the package register
		{
			filename := fmt.Sprintf("pkg/apis/%s/%s/%sRegister.go", strings.Replace(group, ".", "", -1), proto.GetPackage(), strings.Replace(path.Base(proto.GetName()), ".proto", "", -1))
			types, err := gotemplate.New("Types").Funcs(template.FuncMap).Parse(template.REGISTER_TYPES_TEMPLATE)
			if err != nil {
				return err
			}
			err = c.runTemplate(filename, types, k8stypes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *controllerGenerator) runTemplate(filename string, tpl *gotemplate.Template, tpldata interface{}) error {

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	err := tpl.Execute(writer, &tpldata)
	if err != nil {
		return err
	}
	writer.Flush()
	content := buf.Bytes()
	//TODO: Check for .go extension
	if formatted, err := format.Source([]byte(content)); err == nil {
		//We can format this
		content = formatted
	}
	fileContent := string(content)
	var file plugin.CodeGeneratorResponse_File
	file.Name = &filename
	file.Content = &fileContent
	println(fmt.Sprintf("Generated: %s", filename))
	c.Response.File = append(c.Response.File, &file)
	return nil
}

func (c *controllerGenerator) getLocationMessage() map[string][]*LocationMessage {

	ret := make(map[string][]*LocationMessage)
	for index, filename := range c.Request.FileToGenerate {
		locationMessages := make([]*LocationMessage, 0)
		proto := c.Request.ProtoFile[index]
		desc := proto.GetSourceCodeInfo()
		locations := desc.GetLocation()
		for _, location := range locations {
			comments := strings.Split(location.GetLeadingComments(), "\n")
			for _, comment := range comments {
				if strings.Contains(comment, "k8s.io/apimachinery/pkg/runtime.Object") {
					message := proto.GetMessageType()[location.GetPath()[1]]
					locationMessages = append(locationMessages, &LocationMessage{
						Message:  message,
						Location: location,
						Comments: comments[:len(comments)-1],
					})
				}
			}
		}
		ret[filename] = locationMessages
	}
	return ret
}

func (c *controllerGenerator) generateCobra() error {
	// There was a choice here to enforce that each runtime object was its own controller

	locationMessages := c.getLocationMessage()

	var cobraRootOpts template.CobraRootOpts
	cobraRootOpts.ControllerNames = make([]string, 0)
	for index, filename := range c.Request.FileToGenerate {
		proto := c.Request.ProtoFile[index]
		locationMessage := locationMessages[filename]
		for _, location := range locationMessage {
			for _, comment := range location.Comments {
				if strings.Contains(comment, "k8s.io/apimachinery/pkg/runtime.Object") {
					cobraRootOpts.ControllerNames = append(cobraRootOpts.ControllerNames, location.Message.GetName())
				}
			}
		}
		{
			// Generate the root command
			filename := fmt.Sprintf("cmd/root.go")
			cobraroot, err := gotemplate.New("CobraRoot").Funcs(template.FuncMap).Parse(template.CobraRootTemplate)
			if err != nil {
				return err
			}
			err = c.runTemplate(filename, cobraroot, &cobraRootOpts)
			if err != nil {
				return err
			}
		}
		for _, name := range cobraRootOpts.ControllerNames {
			// Generate each controller command
			var tpl template.TemplateOpts
			tpl.Name = name
			tpl.Package = proto.GetPackage()
			tpl.RepoURL = EXAMPLE_REPO

			filename := fmt.Sprintf("cmd/%s.go", name)
			controller, err := gotemplate.New("Test").Funcs(template.FuncMap).Parse(template.CobraControllerTemplate)
			if err != nil {
				return err
			}
			err = c.runTemplate(filename, controller, &tpl)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
