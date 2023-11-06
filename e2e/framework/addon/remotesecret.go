package addon

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/external-secrets/external-secrets-e2e/framework/addon"
)

var _ addon.Addon = (*RemoteSecretDeployment)(nil)

type RemoteSecretDeployment struct {
	config *addon.Config
	output []byte
}

func NewRemoteSecretDeployment(config *addon.Config) *RemoteSecretDeployment {
	return &RemoteSecretDeployment{}
}

func (r RemoteSecretDeployment) Setup(config *addon.Config) error {
	r.config = config
	return nil
}

func (r RemoteSecretDeployment) Install() error {

	// Prepare the kubectl apply command
	cmd := exec.Command("kubectl", "apply", "-f", "/k8s/deploy/remote-secret-controller.yaml")
	//cmd := exec.Command("kubectl", "apply", "-f", "/Users/skabashn/dev/src/redhat-appstudio/remote-secret/e2e/k8s/deploy/remote-secret-controller.yaml")

	// Execute the kubectl apply command
	output, err := cmd.CombinedOutput()

	// Check for errors
	if err != nil {
		fmt.Printf("Error running kubectl apply: %v\n", err)
		return err
	}

	r.output = output

	return nil
}

func (r RemoteSecretDeployment) Logs() error {
	return errors.New(string(r.output))
}

func (r RemoteSecretDeployment) Uninstall() error {
	// Prepare the kubectl apply command
	cmd := exec.Command("kubectl", "delete", "-f", "/k8s/deploy/remote-secret-controller.yaml")

	// Execute the kubectl apply command
	output, err := cmd.CombinedOutput()

	// Check for errors
	if err != nil {
		fmt.Printf("Error running kubectl delete: %v\n", err)
		return err
	}
	r.output = output
	return nil
}
