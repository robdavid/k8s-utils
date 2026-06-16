package kubeutils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func CheckSudo() bool {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		return false
	}
	home, err := homeDirFor(sudoUser)
	if err == nil {
		os.Setenv("HOME", home)
	}
	return true
}

func homeDirFor(username string) (string, error) {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 7)
		if len(parts) >= 6 && parts[0] == username {
			return parts[5], nil
		}
	}
	return "", fmt.Errorf("user %q not found in /etc/passwd", username)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func LoadConfig() (*rest.Config, *kubernetes.Clientset, error) {
	k3sConfig := "/etc/rancher/k3s/k3s.yaml"

	var cfg *rest.Config
	var err error

	kcEnv := os.Getenv("KUBECONFIG")
	kubeConfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	switch {
	case kcEnv != "" && everyFileExists(strings.Split(kcEnv, ":")):
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
	case kcEnv == "" && fileExists(kubeConfig):
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	case fileExists(k3sConfig):
		cfg, err = loadK3sConfig(k3sConfig)
	default:
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
	}

	if err != nil {
		return nil, nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return cfg, clientset, nil
}

func everyFileExists(paths []string) bool {
	for _, p := range paths {
		if !fileExists(p) {
			return false
		}
	}
	return true
}

func loadK3sConfig(path string) (*rest.Config, error) {
	if os.Geteuid() == 0 {
		return clientcmd.BuildConfigFromFlags("", path)
	}
	fmt.Println("Reading kubeconfig for k3s from protected file")
	cmd := exec.Command("sudo", "cat", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("reading k3s config with sudo: %w", err)
	}
	cfg, err := clientcmd.NewClientConfigFromBytes(out)
	if err != nil {
		return nil, err
	}
	return cfg.ClientConfig()
}

var sealedSecretGVR = schema.GroupVersionResource{
	Group:    "bitnami.com",
	Version:  "v1alpha1",
	Resource: "sealedsecrets",
}

func FindSealedSecrets(client dynamic.Interface) ([]SealedSecretRef, error) {
	list, err := client.Resource(sealedSecretGVR).List(context.Background(), unstructuredListOptions())
	if err != nil {
		return nil, fmt.Errorf("listing sealed secrets: %w", err)
	}
	return extractSealedSecretRefs(list), nil
}

func FindNamespacedSealedSecrets(client dynamic.Interface, namespace string) ([]SealedSecretRef, error) {
	list, err := client.Resource(sealedSecretGVR).
		Namespace(namespace).
		List(context.Background(), unstructuredListOptions())
	if err != nil {
		return nil, fmt.Errorf("listing sealed secrets in %s: %w", namespace, err)
	}
	return extractSealedSecretRefs(list), nil
}

func unstructuredListOptions() metav1.ListOptions {
	return metav1.ListOptions{}
}

type SealedSecretRef struct {
	Namespace string
	Name      string
}

func extractSealedSecretRefs(list *unstructured.UnstructuredList) []SealedSecretRef {
	var result []SealedSecretRef
	for _, item := range list.Items {
		ns := item.GetNamespace()
		name := item.GetName()
		if ns != "" && name != "" {
			result = append(result, SealedSecretRef{Namespace: ns, Name: name})
		}
	}
	return result
}
