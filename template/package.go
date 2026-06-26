package template

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectJavaPackage detects the Java package from Maven pom.xml or Gradle build files.
// Returns the detected package or an empty string if detection fails.
func DetectJavaPackage(dir string) string {
	// Maven: groupId + artifactId
	if pkg := fromPomXML(filepath.Join(dir, "pom.xml")); pkg != "" {
		return strings.ToLower(pkg)
	}
	// Gradle: group property
	if pkg := fromGradle(filepath.Join(dir, "build.gradle")); pkg != "" {
		return strings.ToLower(pkg)
	}
	return ""
}

// DetectKotlinPackage detects the Kotlin package from Gradle or Maven build files.
func DetectKotlinPackage(dir string) string {
	// Gradle Kotlin DSL
	if pkg := fromGradleKts(filepath.Join(dir, "build.gradle.kts")); pkg != "" {
		return strings.ToLower(pkg)
	}
	// Groovy Gradle
	if pkg := fromGradle(filepath.Join(dir, "build.gradle")); pkg != "" {
		return strings.ToLower(pkg)
	}
	// Maven with Kotlin
	if pkg := fromPomXML(filepath.Join(dir, "pom.xml")); pkg != "" {
		return strings.ToLower(pkg)
	}
	return ""
}

var groupRe = regexp.MustCompile(`group\s*[=:]\s*['"]([^'"]+)['"]`)
var pomGroupRe = regexp.MustCompile(`<groupId>\s*([^<]+)\s*</groupId>`)
var artifactRe = regexp.MustCompile(`<artifactId>\s*([^<]+)\s*</artifactId>`)

func fromPomXML(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)

	// Strip <parent>...</parent> block so we don't pick up the parent's groupId
	parentRe := regexp.MustCompile(`(?s)<parent>.*?</parent>`)
	content = parentRe.ReplaceAllString(content, "")

	groupMatch := pomGroupRe.FindStringSubmatch(content)
	if len(groupMatch) < 2 {
		return ""
	}
	groupId := strings.TrimSpace(groupMatch[1])

	artifactMatch := artifactRe.FindStringSubmatch(content)
	if len(artifactMatch) < 2 {
		return groupId
	}
	artifactId := strings.TrimSpace(artifactMatch[1])

	return groupId + "." + artifactId
}

func fromGradle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)

	match := groupRe.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func fromGradleKts(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)

	// In Kotlin DSL: group = "com.example" or group = 'com.example'
	// Also: group("com.example")
	match := groupRe.FindStringSubmatch(content)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}

	// Try: group("com.example")
	grpRe := regexp.MustCompile(`group\(['"]([^'"]+)['"]\)`)
	match = grpRe.FindStringSubmatch(content)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}

	return ""
}

// projectDirName returns a lowercased directory name as a fallback package name.
func projectDirName(dir string) string {
	name := filepath.Base(dir)
	if name == "." || name == "" {
		return "app"
	}
	return strings.ToLower(strings.ReplaceAll(name, "-", ""))
}
