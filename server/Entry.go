package main

import "fmt"


type FeedieEntry struct {
	Title string
	Author string
	Published int64 
	Description string
	Thumbnail string
	Links []FeedieLink
}

func (e FeedieEntry) getHashString() string{
	return e.Title + e.Author + fmt.Sprintf("%d",e.Published)
}

func newEntry (title string, author string, published int64, description string, thumbnail string) *FeedieEntry{
	return &FeedieEntry{
		Title: title,
		Author: author,
		Published: published,
		Description: description,
		Thumbnail: thumbnail,
		Links: []FeedieLink{},
	}
}

func newEmptyEntry () *FeedieEntry{
	return &FeedieEntry{
		Title: "",
		Author: "",
		Published: -1,
		Description: "",
		Thumbnail: "",
		Links: []FeedieLink{},
	}

}



type FeedieLink struct {
	URL string
	Type string
}
