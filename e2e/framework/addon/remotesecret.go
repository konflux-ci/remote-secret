package addon

import (
	"context"
	"fmt"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"os"
)

var _ addon.Addon = (*RemoteSecretDeployment)(nil)

type RemoteSecretDeployment struct {
	config *addon.Config
}

func NewRemoteSecretDeployment(config *addon.Config) *RemoteSecretDeployment {
	return &RemoteSecretDeployment{}
}

func (r RemoteSecretDeployment) Setup(config *addon.Config) error {
	r.config = config
	return nil
}

func (r RemoteSecretDeployment) Install() error {

	// Read and unmarshal the YAML file

	file, err := os.Open("/k8s/deploy/remote-secret-controller.yaml")
	if err != nil {
		fmt.Printf("Error opening the file: %v\n", err)
		return err
	}
	defer file.Close() // Make sure to close the file when don

	kubeConfig, err := restclient.InClusterConfig()
	if err != nil {
		return err
	}
	dynamicClient := dynamic.NewForConfigOrDie(kubeConfig)
	// Create an io.Reader from the file
	yamlData := io.Reader(file)

	// Decode YAML into Unstructured objects
	decoder := kubeyaml.NewYAMLOrJSONDecoder(yamlData, 4096)
	obj := &unstructured.Unstructured{}
	for {
		if err := decoder.Decode(obj); err != nil {
			break
		}
		fmt.Printf("start with %s %s", obj.GetName(), obj.GetObjectKind())
		// Create the Unstructured object in the cluster
		_, err = dynamicClient.Resource(obj.GroupVersionKind().GroupVersion().WithResource(obj.GetKind()+"s")).Namespace(obj.GetNamespace()).Create(context.Background(), obj, metav1.CreateOptions{})
		if err != nil {
			fmt.Printf("Error creating object: %v\n", err)
			return err
		}
		fmt.Printf("Object %s/%s created successfully.\n", obj.GetNamespace(), obj.GetName())
	}

	return err
}

func (r RemoteSecretDeployment) Logs() error {
	//TODO implement me
	panic("implement me")
}

func (r RemoteSecretDeployment) Uninstall() error {
	//TODO implement me
	panic("implement me")
}
