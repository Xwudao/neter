package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateIncrementsVersion(t *testing.T) {
	dir := t.TempDir()

	first, err := Create(dir, "add_users")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if first.Version != 1 {
		t.Fatalf("first version = %d, want 1", first.Version)
	}
	if filepath.Base(first.UpPath) != "000001_add_users.up.sql" {
		t.Fatalf("up path = %s", first.UpPath)
	}

	second, err := Create(dir, "Add Posts")
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("second version = %d, want 2", second.Version)
	}
	if filepath.Base(second.UpPath) != "000002_add_posts.up.sql" {
		t.Fatalf("second up path = %s", second.UpPath)
	}
	if _, err := os.Stat(second.DownPath); err != nil {
		t.Fatalf("down migration missing: %v", err)
	}
}

func TestCreateRequiresName(t *testing.T) {
	if _, err := Create(t.TempDir(), "  "); err == nil {
		t.Fatal("expected error for empty migration name")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Add Users":   "add_users",
		"add-users":   "add_users",
		"add.users":   "add_users",
		"  Add__User": "add_user",
		"123abc":      "123abc",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveDSNPriority(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `
db:
  host: 127.0.0.1
  port: 5432
  username: postgres
  password: secret
  database: app
`)

	// Explicit --dsn wins over env and config.
	o, err := NewOptions(root, "", "postgres://explicit@db/app", "", "")
	if err != nil {
		t.Fatal(err)
	}
	dsn, driver, err := o.ResolveDSN()
	if err != nil {
		t.Fatal(err)
	}
	if dsn != "postgres://explicit@db/app" || driver != "postgres" {
		t.Fatalf("got dsn=%q driver=%q", dsn, driver)
	}

	// Env wins over config.
	t.Setenv("NETER_DSN", "mysql://env@tcp(db:3306)/app")
	o, err = NewOptions(root, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	dsn, driver, err = o.ResolveDSN()
	if err != nil {
		t.Fatal(err)
	}
	if dsn != "mysql://env@tcp(db:3306)/app" || driver != "mysql" {
		t.Fatalf("got dsn=%q driver=%q", dsn, driver)
	}
}

func TestResolveDSNFromConfig(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `
db:
  host: 192.168.1.11
  port: 5432
  username: postgres
  password: p@ss
  database: go-testing
`)

	o, err := NewOptions(root, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	dsn, driver, err := o.ResolveDSN()
	if err != nil {
		t.Fatal(err)
	}
	if driver != "postgres" {
		t.Fatalf("driver = %q, want postgres", driver)
	}
	want := "postgres://postgres:p%40ss@192.168.1.11:5432/go-testing?sslmode=disable"
	if dsn != want {
		t.Fatalf("dsn = %q, want %q", dsn, want)
	}
}

func TestResolveDSNMySQLDialect(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `
db:
  dialect: mysql
  host: 127.0.0.1
  port: 3306
  username: root
  password: 123456
  database: nr-demo
`)

	o, err := NewOptions(root, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	dsn, driver, err := o.ResolveDSN()
	if err != nil {
		t.Fatal(err)
	}
	if driver != "mysql" {
		t.Fatalf("driver = %q, want mysql", driver)
	}
	want := "mysql://root:123456@tcp(127.0.0.1:3306)/nr-demo?parseTime=true&multiStatements=true"
	if dsn != want {
		t.Fatalf("dsn = %q, want %q", dsn, want)
	}
}

func TestNewOptionsDefaults(t *testing.T) {
	root := t.TempDir()
	o, err := NewOptions(root, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if o.Dir != filepath.Join(root, DefaultDir) {
		t.Fatalf("dir = %q", o.Dir)
	}
	if o.ConfigFile != filepath.Join(root, DefaultConfigFile) {
		t.Fatalf("config = %q", o.ConfigFile)
	}
}

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, DefaultConfigFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
