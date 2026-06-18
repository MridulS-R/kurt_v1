package modules

import (
	"os"
	"testing"
)

func TestCloudModule_noEnv(t *testing.T) {
	os.Unsetenv("AWS_PROFILE")
	os.Unsetenv("AWS_DEFAULT_PROFILE")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GCLOUD_PROJECT")
	os.Unsetenv("CLOUDSDK_CORE_PROJECT")
	os.Unsetenv("KUBECONFIG")
	// point HOME to empty temp dir so gcloud config doesn't exist
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", orig)

	_, ok := CloudModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment without cloud env vars")
	}
}

func TestCloudModule_aws(t *testing.T) {
	os.Setenv("AWS_PROFILE", "staging")
	defer os.Unsetenv("AWS_PROFILE")

	seg, ok := CloudModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment with AWS_PROFILE")
	}
	if seg != "aws:staging" {
		t.Errorf("got %q, want aws:staging", seg)
	}
}

func TestCloudModule_aws_default_profile_skipped(t *testing.T) {
	os.Setenv("AWS_PROFILE", "default")
	defer os.Unsetenv("AWS_PROFILE")

	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", orig)

	_, ok := CloudModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment for 'default' AWS profile")
	}
}

func TestCloudModule_gcp(t *testing.T) {
	os.Setenv("GOOGLE_CLOUD_PROJECT", "my-project")
	defer os.Unsetenv("GOOGLE_CLOUD_PROJECT")

	seg, ok := CloudModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment with GOOGLE_CLOUD_PROJECT")
	}
	if seg != "gcp:my-project" {
		t.Errorf("got %q, want gcp:my-project", seg)
	}
}

func TestCloudModule_name(t *testing.T) {
	m := CloudModule{}
	if m.Name() != "cloud" {
		t.Error("wrong module name")
	}
}

func TestPythonModule_name(t *testing.T) {
	m := PythonModule{}
	if m.Name() != "python" {
		t.Error("wrong module name")
	}
}

func TestPythonModule_no_file(t *testing.T) {
	os.Unsetenv("PYENV_VERSION")
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	_, ok := PythonModule{}.Render(Context{})
	if ok {
		t.Error("expected no segment without python version files")
	}
}

func TestPythonModule_pythonVersion_file(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)
	os.Unsetenv("PYENV_VERSION")

	os.WriteFile(dir+"/.python-version", []byte("3.11.4\n"), 0644)
	seg, ok := PythonModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment with .python-version")
	}
	if seg != "py:3.11.4" {
		t.Errorf("got %q, want py:3.11.4", seg)
	}
}

func TestPythonModule_env_var(t *testing.T) {
	os.Setenv("PYENV_VERSION", "3.10.0")
	defer os.Unsetenv("PYENV_VERSION")

	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	seg, ok := PythonModule{}.Render(Context{})
	if !ok {
		t.Fatal("expected segment with PYENV_VERSION")
	}
	if seg != "py:3.10.0" {
		t.Errorf("got %q, want py:3.10.0", seg)
	}
}
