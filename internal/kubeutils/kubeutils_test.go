package kubeutils

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
)

func TestFindSealedSecrets(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecret"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecretList"},
		&unstructured.UnstructuredList{},
	)

	client := dynfake.NewSimpleDynamicClient(scheme,
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "bitnami.com/v1alpha1",
				"kind":       "SealedSecret",
				"metadata": map[string]interface{}{
					"name":      "mysecret",
					"namespace": "default",
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "bitnami.com/v1alpha1",
				"kind":       "SealedSecret",
				"metadata": map[string]interface{}{
					"name":      "other",
					"namespace": "kube-system",
				},
			},
		},
	)

	refs, err := FindSealedSecrets(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d: %v", len(refs), refs)
	}
	if refs[0].Name != "mysecret" || refs[0].Namespace != "default" {
		t.Fatalf("unexpected first ref: %+v", refs[0])
	}
	if refs[1].Name != "other" || refs[1].Namespace != "kube-system" {
		t.Fatalf("unexpected second ref: %+v", refs[1])
	}
}

func TestFindSealedSecretsEmpty(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecret"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecretList"},
		&unstructured.UnstructuredList{},
	)

	client := dynfake.NewSimpleDynamicClient(scheme)
	refs, err := FindSealedSecrets(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(refs))
	}
}

func TestFindNamespacedSealedSecrets(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecret"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecretList"},
		&unstructured.UnstructuredList{},
	)

	client := dynfake.NewSimpleDynamicClient(scheme,
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "bitnami.com/v1alpha1",
				"kind":       "SealedSecret",
				"metadata": map[string]interface{}{
					"name":      "ns-secret",
					"namespace": "myns",
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "bitnami.com/v1alpha1",
				"kind":       "SealedSecret",
				"metadata": map[string]interface{}{
					"name":      "other",
					"namespace": "other-ns",
				},
			},
		},
	)

	refs, err := FindNamespacedSealedSecrets(client, "myns")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
	if refs[0].Name != "ns-secret" || refs[0].Namespace != "myns" {
		t.Fatalf("unexpected ref: %+v", refs[0])
	}
}

func TestFindSealedSecretsSkipsEmptyNames(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecret"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecretList"},
		&unstructured.UnstructuredList{},
	)

	client := dynfake.NewSimpleDynamicClient(scheme,
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "bitnami.com/v1alpha1",
				"kind":       "SealedSecret",
				"metadata": map[string]interface{}{
					"name": "",
					"namespace": "",
				},
			},
		},
	)

	refs, err := FindSealedSecrets(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs for empty metadata, got %d", len(refs))
	}
}
