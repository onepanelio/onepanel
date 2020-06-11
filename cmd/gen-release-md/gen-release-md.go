package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type user struct {
	Login              string `json:"login"`
	URL                string `json:"html_url"`
	AvatarURL          string `json:"avatar_url"`
	ContributionsCount int
}

type label struct {
	Name string `json:"name"`
}

type pullRequest struct {
	URL string `json:"url"`
}

type issue struct {
	Number      int          `json:"number"`
	URL         string       `json:"html_url"`
	Title       string       `json:"title"`
	User        user         `json:"user"`
	PullRequest *pullRequest `json:"pull_request"`
	Labels      []label      `json:"labels"`
}

type milestone struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

const (
	apiPrefix = "https://api.github.com/repos/"
)

var releaseTemplate = `# Documentation
See https://docs.onepanel.ai

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
` + "```" + `

## Windows

Download the [attached executable](https://github.com/onepanelio/core/releases/download/v%s/opctl-windows-amd64.exe), rename it to "opctl" and move it to a folder that is in your PATH environment variable.
`

var repositories = []string{
	"onepanelio/core",
	"onepanelio/core-ui",
	"onepanelio/cli",
	"onepanelio/manifests",
	"onepanelio/core-docs",
}

func getPrefixSection(prefix string) (section string) {
	switch prefix {
	case "feat":
		fallthrough
	case "fix":
		fallthrough
	case "docs":
		section = prefix
	default:
		section = "other"
	}

	return
}

// Parse issues, pulling only PRs and categorize them based on labels
// Print everything as MD that can be copied into release notes
func printMarkDown(issues []*issue, version *string) {
	contributorsMap := make(map[string]user, 0)
	sections := make(map[string]string, 0)

	for _, iss := range issues {
		if iss.PullRequest == nil {
			continue
		}

		parts := strings.Split(iss.Title, ":")
		if len(parts) > 0 {
			if user, ok := contributorsMap[iss.User.Login]; ok {
				user.ContributionsCount += 1
				contributorsMap[iss.User.Login] = user
			} else {
				iss.User.ContributionsCount = 1
				contributorsMap[iss.User.Login] = iss.User
			}
			sections[getPrefixSection(parts[0])] += fmt.Sprintf("- %s ([#%d](%s))\n", iss.Title, iss.Number, iss.URL)
		}
	}

	releaseTemplate := fmt.Sprintf(releaseTemplate, *version, *version, *version)
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
	if sections["other"] != "" {
		fmt.Println("## Other")
		fmt.Println(sections["other"])
	}

	fmt.Println("# Contributors")
	contributors := make([]user, 0)
	for _, contributor := range contributorsMap {
		contributors = append(contributors, contributor)
	}
	sort.Slice(contributors, func(i, j int) bool { return contributors[i].ContributionsCount > contributors[j].ContributionsCount })
	for _, user := range contributors {
		fmt.Println(fmt.Sprintf("- <a href=\"%s\"><img src=\"%s\" width=\"12\"/> <strong>%s</strong></a> %s", user.URL, user.AvatarURL, user.Login, user.Login))
	}
}

func httpGet(url string, username, token *string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if username != nil {
		req.SetBasicAuth(*username, *token)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get milestone by title
func getMilestone(repository string, version, username, token *string) (*milestone, error) {
	url := fmt.Sprintf("%s%s/milestones", apiPrefix, repository)
	res, err := httpGet(url, username, token)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("API rate limit exceeded")
	}

	milestones := make([]*milestone, 0)
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
func getIssues(repository string, milestone *milestone, username, token *string) ([]*issue, error) {
	url := fmt.Sprintf("%s%s/issues?state=closed&direction=asc&milestone=%d", apiPrefix, repository, milestone.Number)
	res, err := httpGet(url, username, token)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("API rate limit exceeded")
	}

	issues := make([]*issue, 0)
	if err = json.NewDecoder(res.Body).Decode(&issues); err != nil {
		return nil, err
	}

	return issues, nil
}

func main() {
	version := flag.String("v", "1.0.0", "Version of release, example: -v=1.0.0")
	username := flag.String("u", "", "GitHub username for request, example: -u=octocat")
	token := flag.String("t", "", "GitHub token for request, example: -t=<token>")

	flag.Parse()

	issues := make([]*issue, 0)
	for _, repository := range repositories {
		mil, err := getMilestone(repository, version, username, token)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}

		iss, err := getIssues(repository, mil, username, token)
		if err != nil {
			return
		}
		issues = append(issues, iss...)
	}

	printMarkDown(issues, version)
}
