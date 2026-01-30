package models

import "time"

type FeedItem struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	URL         string      `json:"url"`
	Source      string      `json:"source"`
	SourceType  string      `json:"sourceType"`
	Author      string      `json:"author,omitempty"`
	Summary     string      `json:"summary,omitempty"`
	Content     string      `json:"content,omitempty"`
	PublishedAt time.Time   `json:"publishedAt"`
	FetchedAt   time.Time   `json:"fetchedAt"`
	Thumbnail   string      `json:"thumbnail,omitempty"`
	Tags        []string    `json:"tags"`
	Engagement  *Engagement `json:"engagement,omitempty"`
}

type Engagement struct {
	Upvotes  int `json:"upvotes,omitempty"`
	Comments int `json:"comments,omitempty"`
}

type SourceInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

type FilterParams struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Source string `json:"source"`
	Tag    string `json:"tag"`
	Search string `json:"search"`
}

type AggregatedResponse struct {
	Items       []FeedItem `json:"items"`
	TotalCount  int        `json:"totalCount"`
	FetchedAt   time.Time  `json:"fetchedAt"`
	SourceCount int        `json:"sourceCount"`
}
