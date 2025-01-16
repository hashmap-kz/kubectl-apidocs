package apidocs

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/openapi"
)

type Options struct {
	genericclioptions.IOStreams
	discovery       discovery.CachedDiscoveryInterface
	mapper          meta.RESTMapper
	schema          openapi.Resources
	openAPIV3Client openapiclient.Client
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
	o.RunApp()
	return nil
}

func (o *Options) Complete(f cmdutil.Factory, args []string) error {
	var err error

	o.discovery, err = f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.mapper, err = f.ToRESTMapper()
	if err != nil {
		return err
	}
	o.schema, err = f.OpenAPISchema()
	if err != nil {
		return err
	}
	o.openAPIV3Client, err = f.OpenAPIV3Client()
	if err != nil {
		return err
	}
	return nil
}

// Copy from https://github.com/kubernetes/kubectl/blob/4f380d07c5e5bb41a037a72c4b35c7f828ba2d59/pkg/cmd/cmd.go#L95-L97
func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
}
