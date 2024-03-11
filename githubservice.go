package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type ResponseParser[U any] func(body []byte) (U, error)

var parseListReleases = func(body []byte) (ListReleases, error) {
	if body == nil {
		return nil, fmt.Errorf("failed to get response")
	}

	var list ListReleases
	jsonErr := json.Unmarshal(body, &list)
	if jsonErr != nil {
		slog.Error(fmt.Sprintf("Failed to unmarshal response: %v ", jsonErr))
		return nil, jsonErr
	}

	return list, nil
}

var createReleaseHistoryURL = func(owner, repo string, releases int) string {
	return baseURL + "/repos/" + owner + "/" + repo + "/releases?per_page=" + fmt.Sprintf("%d", releases)
}

func Get[U any](url string, parseResponse ResponseParser[U]) (*U, error) {
	gitClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create request: %v ", err))
		return nil, err
	}

	req.Header.Set("User-Agent", "spacecount-tutorial")

	res, getErr := gitClient.Do(req)
	if getErr != nil {
		slog.Error(fmt.Sprintf("Failed to get response: %v ", getErr))
		return nil, getErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		slog.Error(fmt.Sprintf("Failed to read response: %v ", readErr))
		return nil, readErr
	}

	if res.StatusCode != 200 {
		slog.Error(fmt.Sprintf("Failed to get response: %v ", res.Status))
		return nil, fmt.Errorf("Failed to get response: %v ", res.Status)
	}

	response, parseErr := parseResponse(body)
	if parseErr != nil {
		slog.Error(fmt.Sprintf("Failed to parse response: %v ", parseErr))
		return nil, parseErr
	}

	return &response, nil
}
