package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/perdasilva/crd-update-checker/internal/crdupgradesafety"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var oldCRDPath string
	var newCRDPath string
	var kubeconfig string

	// Set default kubeconfig path
	if kubeconfig = os.Getenv("KUBECONFIG"); kubeconfig == "" {
		if home := os.Getenv("HOME"); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to the kubeconfig file (optional)")
	flag.StringVar(&oldCRDPath, "oldCRD", "", "Path to the old CRD file")
	flag.StringVar(&newCRDPath, "newCRD", "", "Path to the new CRD file")
	flag.Parse()

	// Verify that parameters are set
	if oldCRDPath == "" || newCRDPath == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Both oldCRD and newCRD parameters must be set.\n")
		flag.Usage()
		os.Exit(1)
	}

	// Verify that files exist
	if _, err := os.Stat(oldCRDPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintf(os.Stderr, "File %s does not exist.\n", oldCRDPath)
		os.Exit(1)
	}
	if _, err := os.Stat(newCRDPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintf(os.Stderr, "File %s does not exist.\n", newCRDPath)
		os.Exit(1)
	}

	// Load the files into apiextensions CustomResourceDefinition structs
	oldCRD, err := loadCRDFromFile(oldCRDPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error loading oldCRD: %v\n", err)
		os.Exit(1)
	}

	newCRD, err := loadCRDFromFile(newCRDPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error loading newCRD: %v\n", err)
		os.Exit(1)
	}

	// Get the kubeconfig
	config, err := getKubeConfig(kubeconfig)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error getting kubeconfig: %v\n", err)
		os.Exit(1)
	}

	aeClient, err := apiextensionsv1client.NewForConfig(config)
	if err != nil {
		os.Exit(1)
	}

	checker := crdupgradesafety.NewCRDUpgradeChecker(aeClient.CustomResourceDefinitions())

	if err := checker.Check(ctx, oldCRD, newCRD); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "CRDs are not compatible for upgrade: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully loaded oldCRD and newCRD.")
}

func loadCRDFromFile(path string) (*v1.CustomResourceDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	crd := &v1.CustomResourceDefinition{}

	// Use YAML decoder to decode the YAML into JSON and then into the CRD struct
	yamlDecoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)

	err = yamlDecoder.Decode(crd)
	if err != nil {
		return nil, err
	}

	return crd, nil
}

func getKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		// Use the kubeconfig file provided via the flag
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig path %s: %v", kubeconfigPath, err)
		}
	} else {
		// Throw an error if the kubeconfig file is not found
		return nil, fmt.Errorf("kubeconfig file not found. Please set the KUBECONFIG environment variable or use the --kubeconfig flag")
	}

	return config, nil
}
