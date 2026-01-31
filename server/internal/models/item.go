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
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	SourceType  string `json:"sourceType"`
	Description string `json:"description"`
	FeedType    string `json:"feedType"`
	Enabled     bool   `json:"enabled"`
}

type FilterParams struct {
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
	Sources    []string `json:"sources"`
	SourceType string   `json:"sourceType"`
	Query      string   `json:"query"`
	Sort       string   `json:"sort"`
	FromDate   string   `json:"fromDate"`
	ToDate     string   `json:"toDate"`
	Tag        string   `json:"tag"`
}

type AggregatedResponse struct {
	Items       []FeedItem `json:"items"`
	TotalCount  int        `json:"totalCount"`
	FetchedAt   time.Time  `json:"fetchedAt"`
	SourceCount int        `json:"sourceCount"`
}
