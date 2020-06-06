package main

import (
	"encoding/json"
	"errors"
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

type Milestone struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

const (
	apiPrefix = "https://api.github.com/repos/"
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
		if iss.PullRequest == nil {
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
	fmt.Println("# Changelog")
	if sections["feat"] != "" {
		fmt.Println("## Features")
		fmt.Println(sections["feat"])
	}
	if sections["fix"] != "" {
		fmt.Println("## Fixes")
		fmt.Println(sections["fix"])
	}
	if sections["docs"] != "" {
		fmt.Println("## Docs")
		fmt.Println(sections["docs"])
	}
	if sections["chore"] != "" {
		fmt.Println("## Chores")
		fmt.Println(sections["chore"])
	}

	fmt.Println("# Contributors")
	for _, user := range contributors {
		fmt.Println(fmt.Sprintf("- <a href=\"%s\"><img src=\"%s\" width=\"12\"/> <strong>%s</strong></a> %s", user.URL, user.AvatarURL, user.Login, user.Login))
	}
}

func httpGet(url string, username *string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if username != nil {
		req.SetBasicAuth(*username, "")
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get milestone by title
func getMilestone(repository string, version, username *string) (*Milestone, error) {
	url := fmt.Sprintf("%s%s/milestones", apiPrefix, repository)
	res, err := httpGet(url, username)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	milestones := make([]*Milestone, 0)
	if err = json.NewDecoder(res.Body).Decode(&milestones); err != nil {
		return nil, err
	}

	for _, milestone := range milestones {
		if milestone.Title == "v"+*version {
			return milestone, nil
		}
	}

	return nil, errors.New("milestone not found")
}

// Get issues from repository
func getIssues(repository string, milestone *Milestone, username *string) ([]*Issue, error) {
	url := fmt.Sprintf("%s%s/issues?state=closed&milestone=%d", apiPrefix, repository, milestone.Number)
	res, err := httpGet(url, username)
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
	version := flag.String("v", "0.11.0", "Version of release, example: -v=1.0.0")
	username := flag.String("u", "", "GitHub username for request, example: -u=octocat")

	flag.Parse()

	issues := make([]*Issue, 0)
	for _, repository := range repositories {
		mil, err := getMilestone(repository, version, username)
		if err != nil {
			return
		}

		iss, err := getIssues(repository, mil, username)
		if err != nil {
			return
		}
		issues = append(issues, iss...)
	}

	printMarkDown(issues, version)
}
