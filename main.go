package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/openapi"
)

type Options struct {
	// User input
	apiVersion       string
	inputFieldPath   string
	disablePrintPath bool
	showBrackets     bool

	// After completion
	inputFieldPathRegex *regexp.Regexp
	gvrs                []schema.GroupVersionResource

	// Dependencies
	genericclioptions.IOStreams
	discovery             discovery.CachedDiscoveryInterface
	mapper                meta.RESTMapper
	schema                openapi.Resources
	cachedOpenAPIV3Client openapiclient.Client
}

type groupVersionAPIResource struct {
	schema.GroupVersionResource
	metav1.APIResource
}

func (o *Options) discover() (map[string]*groupVersionAPIResource, []schema.GroupVersionResource, error) {
	lists, err := o.discovery.ServerPreferredResources()
	if err != nil {
		return nil, nil, err
	}
	var gvrs []schema.GroupVersionResource
	m := make(map[string]*groupVersionAPIResource)
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			gvr := gv.WithResource(resource.Name)
			gvrs = append(gvrs, gvr)
			r := groupVersionAPIResource{
				GroupVersionResource: gvr,
				APIResource:          resource,
			}
			m[resource.Name] = &r
			m[resource.Kind] = &r
			m[resource.SingularName] = &r
			for _, shortName := range resource.ShortNames {
				m[shortName] = &r
			}
		}
	}
	sort.SliceStable(gvrs, func(i, j int) bool {
		return gvrs[i].String() < gvrs[j].String()
	})
	return m, gvrs, nil
}

func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		IOStreams: streams,
	}
}

func start() error {
	o := NewOptions(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	kubeConfigFlags := defaultConfigFlags().WithWarningPrinter(o.IOStreams)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	discovery, err := f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.discovery = discovery

	o.mapper, err = f.ToRESTMapper()
	if err != nil {
		return err
	}
	o.schema, err = f.OpenAPISchema()
	if err != nil {
		return err
	}

	c, err := f.OpenAPIV3Client()
	if err != nil {
		return err
	}
	fmt.Print(c)

	a, b, err := o.discover()
	if err != nil {
		return err
	}
	fmt.Println(a, b)
	return nil
}

func main() {
	start()
}

// Copy from https://github.com/kubernetes/kubectl/blob/4f380d07c5e5bb41a037a72c4b35c7f828ba2d59/pkg/cmd/cmd.go#L95-L97
func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
}
