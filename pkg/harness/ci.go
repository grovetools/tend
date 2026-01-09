package harness

import (
	"fmt"
	"os"
	"time"
)

// CIProvider represents different CI environments
type CIProvider string

const (
	CIProviderUnknown       CIProvider = "unknown"
	CIProviderGitHubActions CIProvider = "github-actions"
	CIProviderJenkins       CIProvider = "jenkins"
	CIProviderCircleCI      CIProvider = "circleci"
	CIProviderGitLab        CIProvider = "gitlab"
)

// DetectCIProvider detects the current CI environment
func DetectCIProvider() CIProvider {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return CIProviderGitHubActions
	}
	if os.Getenv("JENKINS_URL") != "" {
		return CIProviderJenkins
	}
	if os.Getenv("CIRCLECI") == "true" {
		return CIProviderCircleCI
	}
	if os.Getenv("GITLAB_CI") == "true" {
		return CIProviderGitLab
	}
	if os.Getenv("CI") == "true" {
		return CIProviderUnknown
	}
	return CIProviderUnknown
}

// IsCI returns true if running in any CI environment
func IsCI() bool {
	return os.Getenv("CI") == "true" || DetectCIProvider() != CIProviderUnknown
}

// ConfigureForCI adjusts options for CI environment
func ConfigureForCI(opts *Options) {
	if !IsCI() {
		return
	}

	// Force non-interactive mode in CI
	opts.Interactive = false

	// Enable verbose output for better debugging
	opts.Verbose = true

	// Continue on error to get full test results
	opts.ContinueOnError = true

	// Adjust timeouts for CI (usually more generous)
	if opts.Timeout < 30*time.Minute {
		opts.Timeout = 30 * time.Minute
	}
}

// SetupCIEnvironment configures environment for CI
func SetupCIEnvironment() {
	provider := DetectCIProvider()

	switch provider {
	case CIProviderGitHubActions:
		// GitHub Actions specific setup
  ulog.Info("::notice::Grove Tend tests starting").Pretty("::notice::Grove Tend tests starting").PrettyOnly().Emit()

	case CIProviderJenkins:
		// Jenkins specific setup
		if workspace := os.Getenv("WORKSPACE"); workspace != "" {
			os.Setenv("GROVE_TEND_WORKSPACE", workspace)
		}

	case CIProviderCircleCI:
		// CircleCI specific setup
		if artifacts := os.Getenv("CIRCLE_ARTIFACTS"); artifacts != "" {
			os.Setenv("GROVE_TEND_ARTIFACTS", artifacts)
		}
	}
}

// GetCIMetadata returns CI-specific metadata
func GetCIMetadata() map[string]string {
	metadata := make(map[string]string)
	provider := DetectCIProvider()

	metadata["ci_provider"] = string(provider)
	metadata["ci"] = fmt.Sprintf("%v", IsCI())

	switch provider {
	case CIProviderGitHubActions:
		metadata["build_id"] = os.Getenv("GITHUB_RUN_ID")
		metadata["build_number"] = os.Getenv("GITHUB_RUN_NUMBER")
		metadata["commit"] = os.Getenv("GITHUB_SHA")
		metadata["branch"] = os.Getenv("GITHUB_REF_NAME")
		metadata["repository"] = os.Getenv("GITHUB_REPOSITORY")
		metadata["actor"] = os.Getenv("GITHUB_ACTOR")

	case CIProviderJenkins:
		metadata["build_id"] = os.Getenv("BUILD_ID")
		metadata["build_number"] = os.Getenv("BUILD_NUMBER")
		metadata["job_name"] = os.Getenv("JOB_NAME")
		metadata["node_name"] = os.Getenv("NODE_NAME")

	case CIProviderCircleCI:
		metadata["build_number"] = os.Getenv("CIRCLE_BUILD_NUM")
		metadata["commit"] = os.Getenv("CIRCLE_SHA1")
		metadata["branch"] = os.Getenv("CIRCLE_BRANCH")
		metadata["job"] = os.Getenv("CIRCLE_JOB")
	}

	return metadata
}