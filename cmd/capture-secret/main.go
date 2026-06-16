package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/robdavid/k8s-utils/internal/kubeutils"
	"github.com/robdavid/k8s-utils/internal/pass"
)

func main() {
	if kubeutils.CheckSudo() {
		fmt.Fprintf(os.Stderr, "This script should not be run as root\n")
		os.Exit(1)
	}

	var allSealed bool
	var listSealed bool
	flag.BoolVar(&allSealed, "all-sealed", false, "Save all secrets backed by a sealed secret")
	flag.BoolVar(&allSealed, "a", false, "Save all secrets backed by a sealed secret")
	flag.BoolVar(&listSealed, "list-sealed", false, "List all sealed secrets")
	flag.BoolVar(&listSealed, "l", false, "List all sealed secrets")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: capture-secret [flags] [namespace[/name] ...]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()

	cfg, clientset, err := kubeutils.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating dynamic client: %v\n", err)
		os.Exit(1)
	}

	store := pass.New()

	switch {
	case allSealed:
		refs, err := kubeutils.FindSealedSecrets(dynClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding sealed secrets: %v\n", err)
			os.Exit(1)
		}
		for _, ref := range refs {
			fmt.Printf("\n%s/%s\n", ref.Namespace, ref.Name)
			if err := saveSecret(clientset, store, ref.Namespace, ref.Name); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving secret %s/%s: %v\n", ref.Namespace, ref.Name, err)
				os.Exit(1)
			}
		}
	case len(args) > 0:
		for _, arg := range args {
			res, err := pass.ParseResource(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing %q: %v\n", arg, err)
				os.Exit(1)
			}
			if res.Name != "" {
				if err := saveSecret(clientset, store, res.Namespace, res.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving secret %s/%s: %v\n", res.Namespace, res.Name, err)
					os.Exit(1)
				}
			} else {
				refs, err := kubeutils.FindNamespacedSealedSecrets(dynClient, res.Namespace)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error finding sealed secrets in %s: %v\n", res.Namespace, err)
					os.Exit(1)
				}
				for _, ref := range refs {
					if err := saveSecret(clientset, store, ref.Namespace, ref.Name); err != nil {
						fmt.Fprintf(os.Stderr, "Error saving secret %s/%s: %v\n", ref.Namespace, ref.Name, err)
						os.Exit(1)
					}
				}
			}
		}
	case listSealed:
		refs, err := kubeutils.FindSealedSecrets(dynClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding sealed secrets: %v\n", err)
			os.Exit(1)
		}
		for _, ref := range refs {
			fmt.Printf("%s/%s\n", ref.Namespace, ref.Name)
		}
	default:
		fmt.Fprintf(os.Stderr, "no secrets given\n")
		os.Exit(1)
	}
}

func saveSecret(clientset kubernetes.Interface, store *pass.Store, namespace, name string) error {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("reading secret %s/%s: %w", namespace, name, err)
	}
	for key, value := range secret.Data {
		val := string(value)
		fmt.Printf("%s/%s/%s (%d chars)\n", namespace, name, key, len(val))
		if err := store.InsertSecret(namespace, name, key, val); err != nil {
			return err
		}
	}
	return nil
}
