package main

type FeedieFeed struct{
	Title string
	Url string
	Entries []FeedieEntry
}

func newFeed(title string, url string, entries []FeedieEntry) *FeedieFeed{
	return &FeedieFeed{
		Title: title,
		Url: url,
		Entries: entries,
	}
}
