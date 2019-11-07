package template

import (
	"strings"
	gotemplate "text/template"
)

type TemplateOpts struct {
	Name        string
	Group       string
	Package     string
	RepoURL     string
	RuntimeType string
}

var FuncMap = gotemplate.FuncMap{
	"ToUpper": strings.ToUpper,
	"ToLower": strings.ToLower,
	"PackageName": func(input string) string {
		return strings.Replace(input, ".", "", -1)
	},
}

var ControllerTemplate = `package controller

import (
	"fmt"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcore{{ .Package }} "k8s.io/client-go/kubernetes/typed/core/{{ .Package }}"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	pb "{{ .RepoURL }}/pkg/apis/{{ .Group | ToLower }}/{{ .Package }}"
	informers "{{ .RepoURL }}/pkg/client/informers/externalversions/{{ .Group | ToLower }}/{{ .Package }}"
	{{ .Name | ToLower}}scheme "{{ .RepoURL }}/pkg/client/clientset/versioned/scheme"
	clientset "{{ .RepoURL }}/pkg/client/clientset/versioned"
)

type {{ .Name | ToLower }}Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeClientset *kubernetes.Clientset
	// {{ .Package | ToLower }}Clientset is our generated clientset
	{{ .Package | ToLower }}Clientset *clientset.Clientset

	informer cache.SharedIndexInformer
	// Controller responsible for processing the FIFO queue of SnapshotPolicy objects
	// and calling provided hook functions
	controller cache.Controller
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	updateQueue workqueue.RateLimitingInterface
	deleteQueue workqueue.RateLimitingInterface
}

func New{{ .Name }}Controller(config *rest.Config) *{{ .Name | ToLower }}Controller {

	utilruntime.Must({{ .Name | ToLower }}scheme.AddToScheme(scheme.Scheme))
	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	{{ .Package | ToLower }}Clientset, err := clientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building cxapi clientset: %s", err.Error())
	}

	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcore{{ .Package }}.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, core{{ .Package }}.EventSource{Component: "{{ .Name }}-operator"})
	resyncPeriod := time.Minute * 1

	controller := &{{ .Name | ToLower }}Controller{
		kubeClientset:  kubeClientset,
		{{ .Package | ToLower }}Clientset: {{ .Package | ToLower }}Clientset,
		recorder:       recorder,
	}

	controller.informer = informers.New{{ .Name }}Informer(
		{{ .Package | ToLower }}Clientset,
		corev1.NamespaceAll,
		resyncPeriod,
		cache.Indexers{})

	controller.informer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.update{{ .Name }},
		UpdateFunc: func(oldObj, newObj interface{}) {
			new{{.Name}} := newObj.(*pb.{{ .Name }})
			old{{.Name}} := oldObj.(*pb.{{ .Name }})
			if new{{.Name}}.ResourceVersion == old{{.Name}}.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.update{{ .Name }}(newObj)
		},
		DeleteFunc: controller.delete{{ .Name }},
	},
		resyncPeriod,
	)

	controller.updateQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "{{ .Name }}Update")
	controller.deleteQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "{{ .Name }}Delete")

	return controller
}

func (c *{{ .Name | ToLower }}Controller) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()

	go c.informer.Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, c.informer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting {{ .Name }} controller")
	println("Starting {{ .Name }} controller")

	// any update context will error out if the {{ .RuntimeType }} delete is ran during create
	go wait.Until(c.runUpdateWorker, time.Second, stopCh)
	go wait.Until(c.runDeleteWorker, time.Second, stopCh)
	<-stopCh

	return nil
}

func (c *{{ .Name | ToLower }}Controller) runUpdateWorker() {
	for c.processNextUpdate() {
	}
}
func (c *{{ .Name | ToLower }}Controller) runDeleteWorker() {
	for c.processNextDelete() {
	}
}

func (c *{{ .Name | ToLower }}Controller) processNextDelete() bool {
	obj, shutdown := c.deleteQueue.Get()

	if shutdown {
		return false
	}

	println("processing delete")

	//We've ensured that anything added to the queue is of type {{ .RuntimeType }}
	objImpl := obj.(*pb.{{ .Name }})

	err := func(objImpl *pb.{{ .Name }}) error {
		defer c.deleteQueue.Done(obj)

		err := c.purge{{ .Name }}(objImpl)
		if err != nil {
			c.deleteQueue.AddRateLimited(objImpl)
			return err
		}
		return nil
	}(objImpl)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *{{ .Name | ToLower }}Controller) processNextUpdate() bool {
	obj, shutdown := c.updateQueue.Get()

	if shutdown {
		return false
	}

	println("processing update")

	//We've ensured that anything added to the queue is of type {{ .Name }}
	objImpl := obj.(*pb.{{ .Name }})

	err := func(objImpl *pb.{{ .Name }}) error {
		defer c.updateQueue.Done(obj)

		err := c.reconcile{{ .Name }}(objImpl)
		if err != nil {
			c.updateQueue.AddRateLimited(objImpl)
			return err
		}
		return nil
	}(objImpl)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *{{ .Name | ToLower }}Controller) update{{ .Name }}(newObj interface{}) {
	if ig, ok := newObj.(*pb.{{ .Name }}); ok {
		c.updateQueue.Add(ig)
	}
}

func (c *{{ .Name | ToLower }}Controller) delete{{ .Name }}(obj interface{}) {
	if ig, ok := obj.(*pb.{{ .Name }}); ok {
		c.deleteQueue.Add(ig)
	}
}

func (c *{{ .Name | ToLower }}Controller) reconcile{{ .Name }}({{ .Name | ToLower }} *pb.{{ .Name }}) error {
	//TODO: Implement
	return fmt.Errorf("reconcile{{ .Name }} not implemented!")
}

func (c *{{ .Name | ToLower }}Controller) purge{{ .Name }}({{ .Name | ToLower }} *pb.{{ .Name }}) error {
	//TODO: Implement
	return fmt.Errorf("delete{{ .Name }} not implemented!")
}
`

var ControllerEntrypoint = `package controller

import (
	"{{ .RepoURL }}/pkg/signals"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

type {{.Name}}Opts struct {
	MasterURL  string
	Kubeconfig string
}

func (opts *{{.Name}}Opts) Run() {

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(opts.MasterURL, opts.Kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	{{ .Name | ToLower }}Controller := New{{ .Name }}Controller(cfg)

	if err = {{ .Name | ToLower }}Controller.Run(stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

}
`
