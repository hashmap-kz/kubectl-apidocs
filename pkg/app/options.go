package app

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kube-openapi/pkg/util/proto"
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

func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		IOStreams: streams,
	}
}

func NewCmd() *cobra.Command {
	o := NewOptions(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	cmd := &cobra.Command{
		Use:   "kubectl apidocs",
		Short: "API resources explained in a tree view format.",
		Example: `
kubectl apidocs
`,
	}
	kubeConfigFlags := defaultConfigFlags().WithWarningPrinter(o.IOStreams)
	flags := cmd.PersistentFlags()
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(flags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	cmd.Run = func(_ *cobra.Command, args []string) {
		cmdutil.CheckErr(o.Complete(f, args))
		cmdutil.CheckErr(o.Run())
	}
	return cmd
}

func (o *Options) Run() error {
	gvarMap, gvrs, err := o.discover()
	if err != nil {
		return err
	}

	for _, gvrsItem := range gvrs {
		if gvar, ok := gvarMap[gvrsItem.Resource]; ok {
			o.inputFieldPathRegex = regexp.MustCompile(".*")
			o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
		}
	}

	// if gvar, ok := gvarMap["statefulsets"]; ok {
	// 	o.inputFieldPathRegex = regexp.MustCompile(".*")
	// 	o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	// }
	// if gvar, ok := gvarMap["httproutes"]; ok {
	// 	o.inputFieldPathRegex = regexp.MustCompile(".*")
	// 	o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	// }
	// if gvar, ok := gvarMap["gateways"]; ok {
	// 	o.inputFieldPathRegex = regexp.MustCompile(".*")
	// 	o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	// }

	pathExplainers := make(map[string]Explainer)
	var paths []path
	for _, gvr := range o.gvrs {
		visitor := &schemaVisitor{
			pathSchema: make(map[path]proto.Schema),
			prevPath: path{
				original:     strings.ToLower(gvr.Resource),
				withBrackets: strings.ToLower(gvr.Resource),
			},
			err: nil,
		}
		gvk, err := o.mapper.KindFor(gvr)
		if err != nil {
			return fmt.Errorf("get the group version kind: %w", err)
		}
		s := o.schema.LookupResource(gvk)
		if s == nil {
			return fmt.Errorf("no schema found for %s", gvk)
		}
		s.Accept(visitor)
		if visitor.err != nil {
			return visitor.err
		}
		filteredPaths := visitor.listPaths(func(s path) bool {
			return o.inputFieldPathRegex.MatchString(s.original)
		})
		for _, p := range filteredPaths {
			pathExplainers[p.original] = Explainer{
				gvr:                 gvr,
				openAPIV3Client:     o.cachedOpenAPIV3Client,
				enablePrintPath:     !o.disablePrintPath,
				enablePrintBrackets: o.showBrackets,
			}
			paths = append(paths, p)
		}
	}

	// Create root node
	root := &Node{Name: "root"}

	// Add each path to the tree
	for _, line := range paths {
		root.AddPath(line.original)
	}

	// UI
	App(root, pathExplainers)
	return nil
}

func (o *Options) Complete(f cmdutil.Factory, args []string) error {
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

	// TODO: cached
	o.cachedOpenAPIV3Client, err = f.OpenAPIV3Client()
	if err != nil {
		return err
	}

	return nil
}

// Copy from https://github.com/kubernetes/kubectl/blob/4f380d07c5e5bb41a037a72c4b35c7f828ba2d59/pkg/cmd/cmd.go#L95-L97
func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
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
