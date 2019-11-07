package template

type CobraRootOpts struct {
	Name            string
	ControllerNames []string
}

var CobraRootTemplate = `package main

import (
	goflag "flag"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootLong  = "Generated {{ .Name }} K8s Controller"
	rootShort = "Generated {{ .Name }} Kubernetes Controller"
)

type RootCmd struct {
	cobraCommand *cobra.Command
}

var rootCommand = RootCmd{
	cobraCommand: &cobra.Command{
		Use:   "{{ .Name }}Controller",
		Short: rootShort,
		Long:  rootLong,
	},
}

func Execute() {
	goflag.Set("logtostderr", "true")
	goflag.CommandLine.Parse([]string{})
	if err := rootCommand.cobraCommand.Execute(); err != nil {
		log.Fatalf("Exit unsuccessfully with err: %v", err)
	}
}

func init() {
	NewCmdRoot(os.Stdout)
}

func NewCmdRoot(out io.Writer) *cobra.Command {

	cmd := rootCommand.cobraCommand

	{{ range $_, $value := .ControllerNames }}
	cmd.AddCommand(NewCmd{{$value}}Controller(out))
	{{ end }}

	return cmd
}

func main() {
	Execute()
}
`

var CobraControllerTemplate = `package main

import (
	"io"

	"{{ .RepoURL }}/pkg/controller"

	"github.com/spf13/cobra"
)

var (
	{{ .Name | ToLower }}ControllerLong    = "start the controller"
	{{ .Name | ToLower }}ControllerExample = "./{{ .Name }}Controller run"
	{{ .Name | ToLower }}ControllerShort   = "start the controller"
)

func NewCmd{{ .Name }}Controller(out io.Writer) *cobra.Command {
	s := &controller.{{ .Name }}Opts{}

	cmd := &cobra.Command{
		Use:     "run",
		Short:   {{ .Name | ToLower }}ControllerShort,
		Long:    {{ .Name | ToLower }}ControllerLong,
		Example: {{ .Name | ToLower }}ControllerExample,
		Run: func(cmd *cobra.Command, args []string) {
			s.Run()
		},
	}

	//Mostly for local debugging
	cmd.Flags().StringVarP(&s.Kubeconfig, "kubeconfig", "k", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVarP(&s.MasterURL, "master", "m", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	return cmd
}

`
