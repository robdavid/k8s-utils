package pass

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const (
	ansiBlue  = "\x1b[01;34m"
	ansiReset = "\x1b[0m"
)

var ansiTree = "\n" +
	"k8s\n" +
	"└── " + ansiBlue + "amycus" + ansiReset + "\n" +
	"    ├── " + ansiBlue + "davical" + ansiReset + "\n" +
	"    │   ├── " + ansiBlue + "davical-app" + ansiReset + "\n" +
	"    │   │   └── password\n" +
	"    │   └── " + ansiBlue + "davical-dba" + ansiReset + "\n" +
	"    │       └── password\n" +
	"    ├── " + ansiBlue + "default" + ansiReset + "\n" +
	"    │   └── " + ansiBlue + "redis-password" + ansiReset + "\n" +
	"    │       └── password\n" +
	"    ├── " + ansiBlue + "home-assistant" + ansiReset + "\n" +
	"    │   └── " + ansiBlue + "mosquitto-auth" + ansiReset + "\n" +
	"    │       └── HA\n" +
	"    ├── " + ansiBlue + "nextcloud" + ansiReset + "\n" +
	"    │   ├── " + ansiBlue + "admin" + ansiReset + "\n" +
	"    │   │   ├── password\n" +
	"    │   │   └── username\n" +
	"    │   ├── " + ansiBlue + "postgres" + ansiReset + "\n" +
	"    │   │   ├── password\n" +
	"    │   │   ├── postgres-password\n" +
	"    │   │   └── username\n" +
	"    │   └── " + ansiBlue + "redis-password" + ansiReset + "\n" +
	"    │       └── password\n" +
	"    └── " + ansiBlue + "photoprism" + ansiReset + "\n" +
	"        ├── " + ansiBlue + "database" + ansiReset + "\n" +
	"        │   ├── mariadb-password\n" +
	"        │   └── mariadb-root-password\n" +
	"        ├── " + ansiBlue + "photoprism-database" + ansiReset + "\n" +
	"        │   ├── PHOTOPRISM_DATABASE_NAME\n" +
	"        │   ├── PHOTOPRISM_DATABASE_PASSWORD\n" +
	"        │   ├── PHOTOPRISM_DATABASE_SERVER\n" +
	"        │   └── PHOTOPRISM_DATABASE_USER\n" +
	"        └── " + ansiBlue + "ui-password" + ansiReset + "\n" +
	"            └── PHOTOPRISM_ADMIN_PASSWORD\n"

var expectedPaths = []string{
	"k8s/amycus/davical/davical-app/password",
	"k8s/amycus/davical/davical-dba/password",
	"k8s/amycus/default/redis-password/password",
	"k8s/amycus/home-assistant/mosquitto-auth/HA",
	"k8s/amycus/nextcloud/admin/password",
	"k8s/amycus/nextcloud/admin/username",
	"k8s/amycus/nextcloud/postgres/password",
	"k8s/amycus/nextcloud/postgres/postgres-password",
	"k8s/amycus/nextcloud/postgres/username",
	"k8s/amycus/nextcloud/redis-password/password",
	"k8s/amycus/photoprism/database/mariadb-password",
	"k8s/amycus/photoprism/database/mariadb-root-password",
	"k8s/amycus/photoprism/photoprism-database/PHOTOPRISM_DATABASE_NAME",
	"k8s/amycus/photoprism/photoprism-database/PHOTOPRISM_DATABASE_PASSWORD",
	"k8s/amycus/photoprism/photoprism-database/PHOTOPRISM_DATABASE_SERVER",
	"k8s/amycus/photoprism/photoprism-database/PHOTOPRISM_DATABASE_USER",
	"k8s/amycus/photoprism/ui-password/PHOTOPRISM_ADMIN_PASSWORD",
}

func TestParsePassLines(t *testing.T) {
	lines := strings.Split(ansiTree, "\n")
	_, parsed, err := ParsePassLines("", lines, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, expectedPaths) {
		t.Fatalf("got:\n%v\nexpected:\n%v", parsed, expectedPaths)
	}
}

func TestCollectKeys(t *testing.T) {
	lines := strings.Split(ansiTree, "\n")
	_, parsed, err := ParsePassLines("", lines, 0)
	if err != nil {
		t.Fatal(err)
	}
	grouped := CollectKeys(parsed)
	expected := map[string][]string{
		"k8s/amycus/davical/davical-app":               {"password"},
		"k8s/amycus/davical/davical-dba":               {"password"},
		"k8s/amycus/default/redis-password":            {"password"},
		"k8s/amycus/home-assistant/mosquitto-auth":     {"HA"},
		"k8s/amycus/nextcloud/admin":                   {"password", "username"},
		"k8s/amycus/nextcloud/postgres":                {"password", "postgres-password", "username"},
		"k8s/amycus/nextcloud/redis-password":          {"password"},
		"k8s/amycus/photoprism/database":               {"mariadb-password", "mariadb-root-password"},
		"k8s/amycus/photoprism/photoprism-database":    {"PHOTOPRISM_DATABASE_NAME", "PHOTOPRISM_DATABASE_PASSWORD", "PHOTOPRISM_DATABASE_SERVER", "PHOTOPRISM_DATABASE_USER"},
		"k8s/amycus/photoprism/ui-password":            {"PHOTOPRISM_ADMIN_PASSWORD"},
	}
	if !reflect.DeepEqual(grouped, expected) {
		t.Fatalf("got:\n%v\nexpected:\n%v", grouped, expected)
	}
}

func TestParsePassLinesNoAnsi(t *testing.T) {
	input := "\n" +
		"k8s\n" +
		"└── amycus\n" +
		"    ├── default\n" +
		"    │   └── mysecret\n" +
		"    │       └── key\n"

	lines := strings.Split(input, "\n")
	_, parsed, err := ParsePassLines("", lines, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"k8s/amycus/default/mysecret/key"}
	if !reflect.DeepEqual(parsed, expected) {
		t.Fatalf("got %v, expected %v", parsed, expected)
	}
}

func TestParseResource(t *testing.T) {
	tests := []struct {
		input string
		want  Resource
	}{
		{"default", Resource{Namespace: "default"}},
		{"default/my-secret", Resource{Namespace: "default", Name: "my-secret"}},
	}
	for _, tt := range tests {
		got, err := ParseResource(tt.input)
		if err != nil {
			t.Errorf("ParseResource(%q) error: %v", tt.input, err)
		} else if got != tt.want {
			t.Errorf("ParseResource(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseResourceErrors(t *testing.T) {
	_, err := ParseResource("a/b/c")
	if err == nil {
		t.Fatal("expected error for a/b/c")
	}
}

func TestListSecrets(t *testing.T) {
	store := &Store{
		RunPass: func(args []string, stdin io.Reader) (string, string, error) {
			if len(args) != 2 || args[0] != "ls" || args[1] != "k8s/myhost" {
				return "", "", fmt.Errorf("unexpected args: %v", args)
			}
			return "Password Store\n└── test\n", "", nil
		},
	}
	paths, err := store.ListSecrets("k8s/myhost")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"k8s/myhost/test"}
	if !slices.Equal(paths, want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
}

func TestGetSecret(t *testing.T) {
	store := &Store{
		RunPass: func(args []string, stdin io.Reader) (string, string, error) {
			if len(args) != 2 || args[0] != "show" || args[1] != "mypath" {
				return "", "", fmt.Errorf("unexpected args: %v", args)
			}
			return "myvalue\n", "", nil
		},
	}
	val, err := store.GetSecret("mypath")
	if err != nil {
		t.Fatal(err)
	}
	if val != "myvalue" {
		t.Fatalf("got %q, want %q", val, "myvalue")
	}
}

func TestGetSecretMultiLine(t *testing.T) {
	store := &Store{
		RunPass: func(args []string, stdin io.Reader) (string, string, error) {
			return "line1\nline2\n", "", nil
		},
	}
	val, err := store.GetSecret("x")
	if err != nil {
		t.Fatal(err)
	}
	if val != "line1\nline2" {
		t.Fatalf("got %q, want %q", val, "line1\nline2")
	}
}

func TestInsertSecret(t *testing.T) {
	var gotArgs []string
	var gotStdin string
	store := &Store{
		RunPass: func(args []string, stdin io.Reader) (string, string, error) {
			gotArgs = append(gotArgs, args...)
			data, _ := io.ReadAll(stdin)
			gotStdin = string(data)
			return "", "", nil
		},
	}
	err := store.InsertSecret("myns", "mysec", "mykey", "myval")
	if err != nil {
		t.Fatal(err)
	}
	if gotStdin != "myval" {
		t.Fatalf("stdin: got %q, want %q", gotStdin, "myval")
	}
	wantArgs := []string{"insert", "--echo", Root() + "/myns/mysec/mykey"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("args: got %v, want %v", gotArgs, wantArgs)
	}
}

func TestStoreDefaults(t *testing.T) {
	store := New()
	if store.RunPass == nil {
		t.Fatal("RunPass should not be nil")
	}
}
