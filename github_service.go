package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ResponseParser[U any] func(body []byte) (U, error)

var listReleasesURL = func(owner, repo string, releases int) string {
	return baseURL + "/repos/" + owner + "/" + repo + "/releases?per_page=" + fmt.Sprintf("%d", releases)
}

var ListReleasesParser = func(body []byte) (ListReleases, error) {
	if body == nil {
		return nil, fmt.Errorf("failed to get response from GitHub ListReleases API")
	}

	var list ListReleases
	jsonErr := json.Unmarshal(body, &list)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to unmarshal response from Github ListReleases API: %w", jsonErr)
	}

	return list, nil
}

func Get[U any](url string, parseResponse ResponseParser[U]) (*U, error) {
	gitClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w ", err)
	}

	res, getErr := gitClient.Do(req)
	if getErr != nil {
		return nil, fmt.Errorf("failed to get response: %w ", getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("Failed to read response: %v ", readErr)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to get response: %v ", res.Status)
	}

	response, parseErr := parseResponse(body)
	if parseErr != nil {
		return nil, fmt.Errorf("Failed to parse response: %v ", parseErr)
	}

	return &response, nil
}
