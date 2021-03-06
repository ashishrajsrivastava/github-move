package cmd

import (
	"fmt"

	"github.com/IBM/argocd-vault-plugin/pkg/config"
	"github.com/IBM/argocd-vault-plugin/pkg/kube"
	"github.com/IBM/argocd-vault-plugin/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewGenerateCommand initializes the generate command
func NewGenerateCommand() *cobra.Command {
	var configPath, secretName string

	var command = &cobra.Command{
		Use:   "generate <path>",
		Short: "Generate manifests from templates with Vault values",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("<path> argument required to generate manifests")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			files, err := listYamlFiles(path)
			if len(files) < 1 {
				return fmt.Errorf("no YAML files were found in %s", path)
			}
			if err != nil {
				return err
			}

			manifests, errs := readFilesAsManifests(files)
			if len(errs) != 0 {

				// TODO: handle multiple errors nicely
				return fmt.Errorf("could not read YAML files: %s", errs)
			}

			viper := viper.New()
			err = setConfig(secretName, configPath, viper)
			if err != nil {
				return err
			}

			config, err := config.New(viper, utils.DefaultHttpClient())
			if err != nil {
				return err
			}

			backend := config.Backend

			err = utils.CheckExistingToken(config.VaultClient)
			if err != nil {
				err = backend.Login()
				if err != nil {
					return err
				}
			}

			for _, manifest := range manifests {
				// skip empty manifests 
				if len(manifest) == 0 {
					continue
				}

				template, err := kube.NewTemplate(manifest, backend, config.PathPrefix)
				if err != nil {
					return err
				}

				err = template.Replace()
				if err != nil {
					return err
				}

				output, err := template.ToYAML()
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%s---\n", output)
			}

			return nil
		},
	}

	command.Flags().StringVarP(&configPath, "config-path", "c", "", "path to a file containing Vault configuration (YAML, JSON, envfile) to use")
	command.Flags().StringVarP(&secretName, "secret-name", "s", "", "name of a Kubernetes Secret containing Vault configuration data in the argocd namespace of your ArgoCD host (Only available when used in ArgoCD)")
	return command
}
