package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/robdavid/k8s-utils/internal/kubeutils"
	"github.com/robdavid/k8s-utils/internal/pass"
)

func main() {
	var (
		root     string
		all      bool
		allShort bool
		noTrim   bool
	)
	if kubeutils.CheckSudo() {
		fmt.Fprintf(os.Stderr, "This script should not be run as root\n")
		os.Exit(1)
	}

	defaultRoot := pass.Root()
	flag.StringVar(&root, "root", defaultRoot, "The root folder for secrets in this cluster")
	flag.BoolVar(&all, "all", false, "Apply all secrets in the pass database under the root")
	flag.BoolVar(&allShort, "a", false, "Apply all secrets in the pass database under the root")
	flag.BoolVar(&noTrim, "no-trim", false, "Don't right trim secret strings")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: apply-secret [flags] [namespace[/name] ...]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if allShort {
		all = true
	}

	args := flag.Args()

	_, clientset, err := kubeutils.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	store := pass.New()

	switch {
	case len(args) > 0:
		for _, arg := range args {
			res, err := pass.ParseResource(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing %q: %v\n", arg, err)
				os.Exit(1)
			}
			prefix := root + "/" + res.Namespace
			if res.Name != "" {
				prefix += "/" + res.Name
			}
			paths, err := store.ListSecrets(prefix)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing secrets at %s: %v\n", prefix, err)
				os.Exit(1)
			}
			bySecret := pass.CollectKeys(paths)
			for path, keys := range bySecret {
				if path == prefix || strings.HasPrefix(path, prefix+"/") {
					if err := writeSecret(clientset, store, path, keys, noTrim); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing secret %s: %v\n", path, err)
						os.Exit(1)
					}
				}
			}
		}
	case all:
		paths, err := store.ListSecrets(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing secrets at %s: %v\n", root, err)
			os.Exit(1)
		}
		bySecret := pass.CollectKeys(paths)
		for path, keys := range bySecret {
			if err := writeSecret(clientset, store, path, keys, noTrim); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing secret %s: %v\n", path, err)
				os.Exit(1)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "no secrets given\n")
		os.Exit(1)
	}
}

func writeSecret(clientset kubernetes.Interface, store *pass.Store, path string, keys []string, noTrim bool) error {
	parts := strings.Split(path, "/")
	namespace := parts[len(parts)-2]
	name := parts[len(parts)-1]

	data := make(map[string][]byte)
	for _, key := range keys {
		val, err := store.GetSecret(path+"/"+key, noTrim)
		if err != nil {
			return fmt.Errorf("reading key %s: %w", key, err)
		}
		data[key] = []byte(val)
	}

	fmt.Printf("namespace = %s, name = %s, keys = %v\n", namespace, name, keys)

	ctx := context.Background()

	existing, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil && existing != nil {
		patch := []map[string]interface{}{
			{"op": "replace", "path": "/data", "value": data},
			{"op": "replace", "path": "/metadata/ownerReferences", "value": []interface{}{}},
		}
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return fmt.Errorf("marshalling patch: %w", err)
		}
		_, err = clientset.CoreV1().Secrets(namespace).Patch(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: data,
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}
