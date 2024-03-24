package main

import "time"

type PromptContext struct {
	Context []Context
}

type Context struct {
	Body        string
	CreatedDate string
	HtmlURL     string
}

type ClaudeResponse struct {
	Content      []Content `json:"content"`
	ID           string    `json:"id"`
	Model        string    `json:"model"`
	Role         string    `json:"role"`
	StopReason   string    `json:"stop_reason"`
	StopSequence any       `json:"stop_sequence"`
	Type         string    `json:"type"`
	Usage        Usage     `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type Content struct {
	ContentType string `json:"type"`
	Text        string `json:"text"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

//// End Claude/Anthropic types

// Start GitHub API types

type ListReleases struct {
	Releases []Release `json:"releases"`
}

type Release struct {
	URL             string    `json:"url"`
	HTMLURL         string    `json:"html_url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	TarballURL      string    `json:"tarball_url"`
	ZipballURL      string    `json:"zipball_url"`
	ID              int       `json:"id"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Body            string    `json:"body"`
	Draft           bool      `json:"draft"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	PublishedAt     time.Time `json:"published_at"`
	Author          Author    `json:"author"`
	Assets          []Assets  `json:"assets"`
}
type Author struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}
type Uploader struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}
type Assets struct {
	URL                string    `json:"url"`
	BrowserDownloadURL string    `json:"browser_download_url"`
	ID                 int       `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	Label              string    `json:"label"`
	State              string    `json:"state"`
	ContentType        string    `json:"content_type"`
	Size               int       `json:"size"`
	DownloadCount      int       `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Uploader           Uploader  `json:"uploader"`
}

type ModelInput struct {
	Name        string
	ContentType string
	Body        []byte
}

type ClaudeRequest struct {
	Version           string    `json:"anthropic_version"`
	Messages          []Message `json:"messages"`
	MaxTokensToSample int       `json:"max_tokens"`
	System            string    `json:"system"`
}
