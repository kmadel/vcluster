package pro

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/loft-sh/api/v3/pkg/auth"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type LoginCmd struct{}

func NewLoginCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := LoginCmd{}

	loginCmd := &cobra.Command{
		Use:                "login",
		Short:              "Log in to the vcluster.pro server",
		DisableFlagParsing: true,
		RunE:               cmd.RunE,
	}

	return loginCmd
}

func (*LoginCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()
	cobraCmd.SilenceUsage = true

	if len(args) == 0 {
		args = []string{"--help"}
	}

	containsHelp := lo.ContainsBy(args, func(item string) bool {
		return item == "--help" || item == "-h"
	})
	if containsHelp {
		err := pro.RunLoftCli(ctx, "latest", append([]string{"login"}, args...))
		if err != nil {
			return fmt.Errorf("failed to run vcluster pro login: %w", err)
		}

		return nil
	}

	serverURL, err := url.Parse(args[0])
	if err != nil {
		return fmt.Errorf("failed to parse vcluster pro server url: %w", err)
	}

	log.GetInstance().Info("Logging in to vcluster pro server %s", serverURL.String())

	serverURL.Path = "/version"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Configure insecure TLS transport
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{Transport: customTransport}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	version := &auth.Version{}

	err = json.NewDecoder(resp.Body).Decode(&version)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	loftVersion := version.Version
	if !strings.HasPrefix(loftVersion, "v") {
		loftVersion = "v" + loftVersion
	}

	log.GetInstance().Infof("Detected server version: %s", loftVersion)

	args = append([]string{"login"}, args...)
	err = pro.RunLoftCli(ctx, loftVersion, args)
	if err != nil {
		return fmt.Errorf("failed to run vcluster pro login: %w", err)
	}

	config, err := pro.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get vcluster pro config: %w", err)
	}

	config.LastUsedVersion = loftVersion

	err = pro.WriteConfig(config)
	if err != nil {
		return fmt.Errorf("failed to write vcluster pro config: %w", err)
	}

	return nil
}
