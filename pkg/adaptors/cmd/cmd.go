package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/wire"
	"github.com/int128/kubelogin/pkg/adaptors"
	"github.com/int128/kubelogin/pkg/models/kubeconfig"
	"github.com/int128/kubelogin/pkg/usecases"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/xerrors"
	"k8s.io/client-go/util/homedir"
)

// Set provides an implementation and interface for Cmd.
var Set = wire.NewSet(
	wire.Struct(new(Cmd), "*"),
	wire.Bind(new(adaptors.Cmd), new(*Cmd)),
)

const examples = `  # Login to the provider using the authorization code flow.
  %[1]s

  # Login to the provider using the resource owner password credentials flow.
  %[1]s --username USERNAME --password PASSWORD

  # Run as a credential plugin.
  %[1]s get-token --oidc-issuer-url=https://issuer.example.com`

var defaultListenPort = []int{8000, 18000}
var defaultTokenCacheDir = homedir.HomeDir() + "/.kube/cache/oidc-login"

// Cmd provides interaction with command line interface (CLI).
type Cmd struct {
	Login    usecases.Login
	GetToken usecases.GetToken
	Logger   adaptors.Logger
}

// Run parses the command line arguments and executes the specified use-case.
// It returns an exit code, that is 0 on success or 1 on error.
func (cmd *Cmd) Run(ctx context.Context, args []string, version string) int {
	executable := filepath.Base(args[0])

	rootCmd := newRootCmd(ctx, executable, cmd)
	rootCmd.Version = version
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	getTokenCmd := newGetTokenCmd(ctx, cmd)
	rootCmd.AddCommand(getTokenCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			cmd.Logger.Printf("%s version %s", executable, version)
		},
	}
	rootCmd.AddCommand(versionCmd)

	rootCmd.SetArgs(args[1:])
	if err := rootCmd.Execute(); err != nil {
		cmd.Logger.Printf("error: %s", err)
		cmd.Logger.Debugf(1, "stacktrace: %+v", err)
		return 1
	}
	return 0
}

// kubectlOptions represents kubectl specific options.
type kubectlOptions struct {
	Kubeconfig           string
	Context              string
	User                 string
	CertificateAuthority string
	SkipTLSVerify        bool
	Verbose              int
}

func (o *kubectlOptions) register(f *pflag.FlagSet) {
	f.SortFlags = false
	f.StringVar(&o.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	f.StringVar(&o.Context, "context", "", "The name of the kubeconfig context to use")
	f.StringVar(&o.User, "user", "", "The name of the kubeconfig user to use. Prior to --context")
	f.StringVar(&o.CertificateAuthority, "certificate-authority", "", "Path to a cert file for the certificate authority")
	f.BoolVar(&o.SkipTLSVerify, "insecure-skip-tls-verify", false, "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure")
	f.IntVarP(&o.Verbose, "v", "v", 0, "If set to 1 or greater, it shows debug log")
}

// loginOptions represents the options for Login use-case.
type loginOptions struct {
	ListenPort      []int
	SkipOpenBrowser bool
	Username        string
	Password        string
}

func (o *loginOptions) register(f *pflag.FlagSet) {
	f.SortFlags = false
	f.IntSliceVar(&o.ListenPort, "listen-port", defaultListenPort, "Port to bind to the local server. If multiple ports are given, it will try the ports in order")
	f.BoolVar(&o.SkipOpenBrowser, "skip-open-browser", false, "If true, it does not open the browser on authentication")
	f.StringVar(&o.Username, "username", "", "If set, perform the resource owner password credentials grant")
	f.StringVar(&o.Password, "password", "", "If set, use the password instead of asking it")
}

func newRootCmd(ctx context.Context, executable string, cmd *Cmd) *cobra.Command {
	var o struct {
		kubectlOptions
		loginOptions
	}
	rootCmd := &cobra.Command{
		Use:     executable,
		Short:   "Login to the OpenID Connect provider and update the kubeconfig",
		Example: fmt.Sprintf(examples, executable),
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			cmd.Logger.SetLevel(adaptors.LogLevel(o.Verbose))
			in := usecases.LoginIn{
				KubeconfigFilename: o.Kubeconfig,
				KubeconfigContext:  kubeconfig.ContextName(o.Context),
				KubeconfigUser:     kubeconfig.UserName(o.User),
				CACertFilename:     o.CertificateAuthority,
				SkipTLSVerify:      o.SkipTLSVerify,
				ListenPort:         o.ListenPort,
				SkipOpenBrowser:    o.SkipOpenBrowser,
				Username:           o.Username,
				Password:           o.Password,
			}
			if err := cmd.Login.Do(ctx, in); err != nil {
				return xerrors.Errorf("error: %w", err)
			}
			return nil
		},
	}
	o.kubectlOptions.register(rootCmd.Flags())
	o.loginOptions.register(rootCmd.Flags())
	return rootCmd
}

// getTokenOptions represents the options for get-token command.
type getTokenOptions struct {
	loginOptions
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	ExtraScopes          []string
	CertificateAuthority string
	SkipTLSVerify        bool
	Verbose              int
	TokenCacheDir        string
}

func (o *getTokenOptions) register(f *pflag.FlagSet) {
	f.SortFlags = false
	o.loginOptions.register(f)
	f.StringVar(&o.IssuerURL, "oidc-issuer-url", "", "Issuer URL of the provider (mandatory)")
	f.StringVar(&o.ClientID, "oidc-client-id", "", "Client ID of the provider (mandatory)")
	f.StringVar(&o.ClientSecret, "oidc-client-secret", "", "Client secret of the provider")
	f.StringSliceVar(&o.ExtraScopes, "oidc-extra-scope", nil, "Scopes to request to the provider")
	f.StringVar(&o.CertificateAuthority, "certificate-authority", "", "Path to a cert file for the certificate authority")
	f.BoolVar(&o.SkipTLSVerify, "insecure-skip-tls-verify", false, "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure")
	f.IntVarP(&o.Verbose, "v", "v", 0, "If set to 1 or greater, it shows debug log")
	f.StringVar(&o.TokenCacheDir, "token-cache-dir", defaultTokenCacheDir, "Path to a directory for caching tokens")
}

func newGetTokenCmd(ctx context.Context, cmd *Cmd) *cobra.Command {
	var o getTokenOptions
	c := &cobra.Command{
		Use:   "get-token [flags]",
		Short: "Run as a kubectl credential plugin",
		Args: func(c *cobra.Command, args []string) error {
			if err := cobra.NoArgs(c, args); err != nil {
				return err
			}
			if o.IssuerURL == "" {
				return xerrors.New("--oidc-issuer-url is missing")
			}
			if o.ClientID == "" {
				return xerrors.New("--oidc-client-id is missing")
			}
			return nil
		},
		RunE: func(*cobra.Command, []string) error {
			cmd.Logger.SetLevel(adaptors.LogLevel(o.Verbose))
			in := usecases.GetTokenIn{
				IssuerURL:       o.IssuerURL,
				ClientID:        o.ClientID,
				ClientSecret:    o.ClientSecret,
				ExtraScopes:     o.ExtraScopes,
				CACertFilename:  o.CertificateAuthority,
				SkipTLSVerify:   o.SkipTLSVerify,
				ListenPort:      o.ListenPort,
				SkipOpenBrowser: o.SkipOpenBrowser,
				Username:        o.Username,
				Password:        o.Password,
				TokenCacheDir:   o.TokenCacheDir,
			}
			if err := cmd.GetToken.Do(ctx, in); err != nil {
				return xerrors.Errorf("error: %w", err)
			}
			return nil
		},
	}
	o.register(c.Flags())
	return c
}