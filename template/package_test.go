package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── DetectJavaPackage ─────────────────────────────────────────────────────────

func TestDetectJavaPackage_MavenPom(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(`
<project>
    <groupId>com.example</groupId>
    <artifactId>my-service</artifactId>
</project>
`), 0644)

	pkg := DetectJavaPackage(dir)
	if pkg != "com.example.my-service" {
		t.Errorf("DetectJavaPackage = %q, want %q", pkg, "com.example.my-service")
	}
}

func TestDetectJavaPackage_GradleGroup(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(`
plugins {
    id 'java'
}
group = 'org.myapp'
version = '1.0.0'
`), 0644)

	pkg := DetectJavaPackage(dir)
	if pkg != "org.myapp" {
		t.Errorf("DetectJavaPackage = %q, want %q", pkg, "org.myapp")
	}
}

func TestDetectJavaPackage_NoBuildFile(t *testing.T) {
	dir := t.TempDir()
	pkg := DetectJavaPackage(dir)
	if pkg != "" {
		t.Errorf("expected empty, got %q", pkg)
	}
}

func TestDetectJavaPackage_MavenWithColonGroup(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(`
<project>
    <groupId>io.startup</groupId>
    <artifactId>api-gateway</artifactId>
</project>
`), 0644)

	pkg := DetectJavaPackage(dir)
	if pkg != "io.startup.api-gateway" {
		t.Errorf("DetectJavaPackage = %q, want %q", pkg, "io.startup.api-gateway")
	}
}

// ── DetectKotlinPackage ───────────────────────────────────────────────────────

func TestDetectKotlinPackage_GradleKts(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(`
plugins {
    kotlin("jvm")
}
group = "com.kotlin.app"
version = "1.0.0"
`), 0644)

	pkg := DetectKotlinPackage(dir)
	if pkg != "com.kotlin.app" {
		t.Errorf("DetectKotlinPackage = %q, want %q", pkg, "com.kotlin.app")
	}
}

func TestDetectKotlinPackage_GradleKtsGroupFunc(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(`
plugins {
    kotlin("jvm")
}
group("com.func.app")
version("1.0.0")
`), 0644)

	pkg := DetectKotlinPackage(dir)
	if pkg != "com.func.app" {
		t.Errorf("DetectKotlinPackage = %q, want %q", pkg, "com.func.app")
	}
}

func TestDetectKotlinPackage_GradleGroovy(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(`
apply plugin: 'kotlin'
group = 'com.example'
version = '1.0'
`), 0644)

	pkg := DetectKotlinPackage(dir)
	if pkg != "com.example" {
		t.Errorf("DetectKotlinPackage = %q, want %q", pkg, "com.example")
	}
}

func TestDetectKotlinPackage_MavenPom(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(`
<project>
    <groupId>com.example</groupId>
    <artifactId>kotlin-app</artifactId>
</project>
`), 0644)

	pkg := DetectKotlinPackage(dir)
	if pkg != "com.example.kotlin-app" {
		t.Errorf("DetectKotlinPackage = %q, want %q", pkg, "com.example.kotlin-app")
	}
}

func TestDetectKotlinPackage_NoBuildFile(t *testing.T) {
	dir := t.TempDir()
	pkg := DetectKotlinPackage(dir)
	if pkg != "" {
		t.Errorf("expected empty, got %q", pkg)
	}
}

// ── Integration: package appears in generated TOML ────────────────────────────

func TestGetFlashORMConfig_JavaPackageInToml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(`
<project>
    <groupId>com.mycorp</groupId>
    <artifactId>data-layer</artifactId>
</project>
`), 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	pt := NewProjectTemplateExt(PostgreSQL, false, false, false, true)
	if pkg := DetectJavaPackage("."); pkg != "" {
		pt.JavaPackage = pkg
	}

	cfg := pt.GetFlashORMConfig()
	if !strings.Contains(cfg, `package = "com.mycorp.data-layer"`) {
		t.Errorf("generated config missing package field:\n%s", cfg)
	}
}

func TestGetFlashORMConfig_KotlinPackageInToml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(`group = "com.kotlin.app"`), 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	pt := NewProjectTemplateExt(PostgreSQL, false, false, true, false)
	if pkg := DetectKotlinPackage("."); pkg != "" {
		pt.KotlinPackage = pkg
	}

	cfg := pt.GetFlashORMConfig()
	if !strings.Contains(cfg, `package = "com.kotlin.app"`) {
		t.Errorf("generated config missing package field:\n%s", cfg)
	}
}
