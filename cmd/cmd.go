package cmd

import (
	"os"

	"github.com/hashmap-kz/kubectl-apidocs/internal/apidocs"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/openapi"
)

type APIDocsOptions struct {
	genericiooptions.IOStreams
	discoveryClient discovery.CachedDiscoveryInterface
	restMapper      meta.RESTMapper
	openAPISchema   openapi.Resources
	openAPIClient   openapiclient.Client
}

func NewAPIDocsOptions(streams genericiooptions.IOStreams) *APIDocsOptions {
	return &APIDocsOptions{
		IOStreams: streams,
	}
}

func NewCmdAPIDocs() *cobra.Command {
	o := NewAPIDocsOptions(genericiooptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	cmd := &cobra.Command{
		Use:   "kubectl apidocs",
		Short: "API resources explained in a tree view format.",
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

func (o *APIDocsOptions) Run() error {
	err := apidocs.RunApp(&apidocs.UIData{
		DiscoveryClient: o.discoveryClient,
		RestMapper:      o.restMapper,
		OpenAPISchema:   o.openAPISchema,
		OpenAPIClient:   o.openAPIClient,
	})
	return err
}

func (o *APIDocsOptions) Complete(f cmdutil.Factory, _ []string) error {
	var err error
	o.discoveryClient, err = f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.restMapper, err = f.ToRESTMapper()
	if err != nil {
		return err
	}
	o.openAPISchema, err = f.OpenAPISchema()
	if err != nil {
		return err
	}
	o.openAPIClient, err = f.OpenAPIV3Client()
	if err != nil {
		return err
	}
	return nil
}

func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).
		WithDeprecatedPasswordFlag().
		WithDiscoveryBurst(300).
		WithDiscoveryQPS(50.0)
}
