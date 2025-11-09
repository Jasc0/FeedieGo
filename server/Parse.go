package main

import (
	"log"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/mmcdole/gofeed"
)



func parser(url string) *FeedieFeed{
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		log.Printf("unable to parse :%s, %s",url, err)
		return nil
	}
	var items []FeedieEntry
	for _, item := range feed.Items {
		new_entry := newEmptyEntry()
		new_entry.Title = item.Title

		// Authors can be empty; and Author is a *gofeed.Person
		if len(item.Authors) > 0 && item.Authors[0] != nil {
			new_entry.Author = item.Authors[0].Name
		}
		if len(new_entry.Author) == 0{
			new_entry.Author = feed.Title
		}

		pubtime, err := dateparse.ParseAny(item.Published)
		if err != nil{
			if strings.Contains(err.Error(),"hour out of range"){
				pubStr := item.Published
				pubStr = strings.Replace(pubStr,"24:","00:", 1)
				pubtime, err = dateparse.ParseAny(pubStr)
				if err == nil{
					pubtime = pubtime.Add(24 * time.Hour)
				} else{
					pubtime = time.Now()
					log.Printf("unable to parse date %s, %v\n", item.Published , err)
				}
			}
		}
		new_entry.Published = int64(pubtime.Unix())

		new_entry.Description = item.Description
		if len(new_entry.Description) < len(item.Content){
			new_entry.Description = item.Content
		}

		if item.Image != nil{
		new_entry.Thumbnail = item.Image.URL
		}

		for _, l := range item.Links{
			fl := FeedieLink{URL: l, Type: "text/html"}
			new_entry.Links = append(new_entry.Links, fl)
		}
		
		for _, enc := range item.Enclosures{
			fl := FeedieLink{URL: enc.URL, Type: enc.Type}
			new_entry.Links = append(new_entry.Links, fl)
		}


		if item.ITunesExt != nil{
			parseItunes(item,feed,new_entry)
		}
		if item.Extensions != nil{
			parseExtensions(item,feed,new_entry)
		}

		items = append(items, *new_entry)

	}
	new_feed := newFeed(feed.Title, url, items)
	return new_feed
	//for _, i := range items{
	//	fmt.Printf("Title: %s\nAuthor: %s\npub: %d\nthumbnail: %s\nLinks: %v\n description: %s\n", i.Title, i.Author, i.Published, i.Thumbnail, i.Links, i.Description)
	//}

}

func parseItunes(feedItem *gofeed.Item, feedFeed *gofeed.Feed, outItem *FeedieEntry){
	if feedItem.ITunesExt != nil{
		itunes := feedItem.ITunesExt
		if(len(outItem.Description) < len (itunes.Summary)){
			outItem.Description = itunes.Summary
		}
		if(len(outItem.Thumbnail) < len (itunes.Image)){
			outItem.Thumbnail = itunes.Image
		}
		if(len(outItem.Thumbnail) < len (feedFeed.Image.URL)){
			outItem.Thumbnail = feedFeed.Image.URL
		}

	}

}
func parseExtensions(feedItem *gofeed.Item, feedFeed *gofeed.Feed, outItem *FeedieEntry){
	//media
	if media, ok := feedItem.Extensions["media"]; ok {
		if groups, ok := media["group"]; ok && len(groups) > 0 {
			g := groups[0] // the <media:group> element

			// description
			if descs, ok := g.Children["description"]; ok && len(descs) > 0 {
				d := descs[0].Value
				if len(d) > len(outItem.Description){
					outItem.Description = d
				}
			}

			// thumbnail url
			if thumbs, ok := g.Children["thumbnail"]; ok && len(thumbs) > 0 {
				if url, ok := thumbs[0].Attrs["url"]; ok {
					if len(url) > len(outItem.Thumbnail){
						outItem.Thumbnail = url
					}
				}
			}
		}
		if thumbs, ok := media["thumbnail"]; ok && len(thumbs) > 0 {
			outItem.Thumbnail = thumbs[0].Attrs["url"]
	  }
  }

}

