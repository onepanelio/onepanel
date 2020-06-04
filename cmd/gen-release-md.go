package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
)

type User struct {
	Login     string `json:"login"`
	URL       string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

type Label struct {
	Name string `json:"name"`
}

type PullRequest struct {
	URL string `json:"url"`
}

type Issue struct {
	Number      int          `json:"number"`
	URL         string       `json:"html_url"`
	Title       string       `json:"title"`
	User        User         `json:"user"`
	PullRequest *PullRequest `json:"pull_request"`
	Labels      []Label      `json:"labels"`
}

const (
	apiPrefix          = "https://api.github.com/repos/"
	issuesPathAndQuery = "/issues?milestone=3&state=closed"
)

var releaseTemplate = `# Documentation
See [Documentation](https://docs.onepanel.ai)

# CLI Installation

## Linux

` + "```" + `
# Download the binary
curl -sLO https://github.com/onepanelio/core/releases/download/v%s/opctl-linux-amd64

# Make binary executable
chmod +x opctl-linux-amd64

# Move binary to path
mv ./opctl-linux-amd64 /usr/local/bin/opctl

# Test installation
opctl version
` + "```" + `

## macOS

` + "```" + `
# Download the binary
curl -sLO https://github.com/onepanelio/core/releases/download/v%s/opctl-macos-amd64

# Make binary executable
chmod +x opctl-macos-amd64

# Move binary to path
mv ./opctl-macos-amd64 /usr/local/bin/opctl

# Test installation
opctl version
` + "```"

var repositories = []string{
	"onepanelio/core",
	"onepanelio/core-ui",
	"onepanelio/cli",
	"onepanelio/manifests",
	"onepanelio/core-docs",
}

// Parse issues, pulling only PRs and categorize them based on labels
// Print everything as MD that can be copied into release notes
func printMarkDown(issues []*Issue, version *string) {
	contributors := make(map[string]User, 0)
	sections := make(map[string]string, 0)

	for _, iss := range issues {
		if iss.PullRequest == nil || len(iss.Labels) == 0 {
			continue
		}

		parts := strings.Split(iss.Title, ":")
		if len(parts) > 0 {
			contributors[iss.User.Login] = iss.User
			sections[parts[0]] += fmt.Sprintf("- %s ([#%d](%s))\n", iss.Title, iss.Number, iss.URL)
		}
	}

	releaseTemplate := fmt.Sprintf(releaseTemplate, *version, *version)
	fmt.Println(releaseTemplate)
	fmt.Println("# Changelog\n")
	fmt.Println("## Features\n")
	fmt.Println(sections["feat"])
	fmt.Println("## Fixes\n")
	fmt.Println(sections["fix"])
	fmt.Println("## Docs\n")
	fmt.Println(sections["docs"])
	fmt.Println("## Chores\n")
	fmt.Println(sections["chore"])

	fmt.Println("# Contributors\n")
	for _, user := range contributors {
		fmt.Println(fmt.Sprintf("- <a href=\"%s\"><img src=\"%s\" width=\"12\"/> <strong>%s</strong></a> %s", user.URL, user.AvatarURL, user.Login, user.Login))
	}
}

// Get issues from repository
func getIssues(repository string) ([]*Issue, error) {
	res, err := http.Get(apiPrefix + repository + issuesPathAndQuery)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	issues := make([]*Issue, 0)
	if err = json.NewDecoder(res.Body).Decode(&issues); err != nil {
		return nil, err
	}

	return issues, nil
}

func main() {
	version := flag.String("v", "1.0.0", "Version of release, example: -v=1.0.0")

	flag.Parse()

	issues := make([]*Issue, 0)
	for _, repository := range repositories {
		iss, err := getIssues(repository)
		if err != nil {
			return
		}
		issues = append(issues, iss...)
	}

	printMarkDown(issues, version)
}
