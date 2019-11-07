package template

var REGISTER_GROUP_TEMPLATE = `package {{ .Group | PackageName }}

const (
	GroupName = "{{ .Group }}"
)`

var DOC_TEMPLATE = `// +k8s:deepcopy-gen=package

// +groupName={{ .Group }}
package {{ .Package }}`

var DREKLE_NAME_ANNOTATION_KEY string = "+drekle:k8s:name="
var DREKLE_STATUS_TYPE_KEY string = "+drekle:k8s:status="

type ProtoMessage struct {
	Package string
	RepoURL string
	Group   string
	// using the +drekle:k8s:name annotation
	Name            string
	RuntimeType     string
	StatusType      string
	LeadingComments []string
}

type ProtoFile struct {
	Package  string
	Group    string
	RepoURL  string
	Messages []*ProtoMessage
}

var K8S_TYPE_TEMPLATE = `package {{ .Package }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
{{ range $_, $value := .Messages }}
	{{ $value.Name }}Resource = "{{ $value.Name | ToLower }}"
	{{ $value.Name }}ResourcePlural = "{{ $value.Name | ToLower }}s"{{ end }}
)

{{ range $_, $value := .Messages }}
{{ range $_, $comment := $value.LeadingComments }}
// {{ $comment }}{{ end }}
type {{ $value.Name }} struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   {{ $value.RuntimeType }} ` + "`json:\"spec\"`" + `
	{{ if $value.StatusType }}
	Status {{ $value.StatusType }}  ` + "`json:\"status\"`" + `
	{{ end }}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type {{ $value.Name }}List struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Items   []{{ $value.Name }} ` + "`json:\"items\"`" + `
}
{{ end }}
`

var REGISTER_TYPES_TEMPLATE = `
package {{ .Package }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"

	group "{{ .RepoURL }}/pkg/apis/{{ .Group }}"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: group.GroupName, Version: "{{ .Package }}"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// Variables referenced in generation
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		{{ range $_, $message := .Messages }}
		&{{ $message.Name }}{},
		{{ end }}
	)
	{{ range $_, $message := .Messages }}
	scheme.AddKnownTypeWithName(SchemeGroupVersion.WithKind({{ $message.Name }}ResourcePlural), &{{ $message.Name }}List{})
	{{ end }}
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
`
