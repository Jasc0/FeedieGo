package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func getAllFeedEntries(config FeedieConfig, offset int) []list_entry {
	entries := []list_entry{}

	resp, err := http.Get(fmt.Sprintf("%s%s/get_entries?method=all&limit=%d&offset=%d",
		config.SERVER, config.PORT, config.EntryLimit, offset*config.EntryLimit))
	if err != nil {
		log.Println(err)
		return entries
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Bad StatusCode")
		return entries
	}

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		log.Fatal(err)
	}

	return entries
}

func getSrcFunc(srcType SourceType, rawKey string) func(FeedieConfig, int) []list_entry {
	escapedKey := url.QueryEscape(rawKey)

	switch srcType {
	case Tag:
		return func(config FeedieConfig, offset int) []list_entry {
			entries := []list_entry{}
			resp, err := http.Get(fmt.Sprintf("%s%s/get_entries?method=by_tag&value=%s&limit=%d&offset=%d",
				config.SERVER, config.PORT, escapedKey, config.EntryLimit, offset*config.EntryLimit))
			if err != nil {
				log.Println(err)
				return entries
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Println("Bad StatusCode")
				return entries
			}
			if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
				log.Fatal(err)
			}
			return entries
		}
	case Feed:
		return func(config FeedieConfig, offset int) []list_entry {
			entries := []list_entry{}
			resp, err := http.Get(fmt.Sprintf("%s%s/get_entries?method=by_feed&value=%s&limit=%d&offset=%d",
				config.SERVER, config.PORT, escapedKey, config.EntryLimit, offset*config.EntryLimit))
			if err != nil {
				log.Println(err)
				return entries
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Println("Bad StatusCode", resp.StatusCode)
				return entries
			}
			if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
				log.Fatal(err)
			}
			return entries
		}
	}
	return func(config FeedieConfig, offset int) []list_entry {
		_, _ = config, offset
		return []list_entry{}
	}
}

type ActionType int 
const(
	addFeed_t ActionType = iota
	delFeed_t
	addTag_t
	delTag_t
	modTagMember_t
)

func getActionFunc (at ActionType) func (FeedieConfig, []string) error {
	switch(at){
	case addFeed_t:
			return func(config FeedieConfig, params []string) error {
				if len(params) != 1 {return errors.New("Invalid parameter count")}

				resp, err := http.Get(fmt.Sprintf("%s%s/add_feed?feed_url=%s",
				config.SERVER,config.PORT,url.QueryEscape(params[0]))); if err != nil{
					log.Println(err)
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK{
					log.Println("Bad StatusCode", resp.StatusCode)
					return fmt.Errorf("invalid feed url: %s", params[0])
				}

				return nil
			}
	case delFeed_t:
			return func(config FeedieConfig, params []string) error {
				if len(params) != 1 {return errors.New("Invalid parameter count")}

				resp, err := http.Get(fmt.Sprintf("%s%s/del_feed?feed_url=%s",
					config.SERVER,config.PORT,url.QueryEscape(params[0]))); if err != nil{
					log.Println(err)
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK{
					log.Println("Bad StatusCode", resp.StatusCode)
					return fmt.Errorf("invalid feed url: %s", params[0])
				}

				return nil
			}
		case addTag_t:
			return func(config FeedieConfig, params []string) error{
				if len(params) != 1 {return errors.New("Invalid parameter count")}

				resp, err := http.Get(fmt.Sprintf("%s%s/add_tag?tag_name=%s",
					config.SERVER,config.PORT,url.QueryEscape(params[0]))); if err != nil{
					log.Println(err)
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK{
					log.Println("Bad StatusCode", resp.StatusCode)
					return fmt.Errorf("invalid tag: %s", params[0])
				}

				return nil
			}
		case delTag_t:
			return func(config FeedieConfig, params []string) error{
				if len(params) != 1 {return errors.New("Invalid parameter count")}

				resp, err := http.Get(fmt.Sprintf("%s%s/del_tag?tag_name=%s",
					config.SERVER,config.PORT,url.QueryEscape(params[0]))); if err != nil{
					log.Println(err)
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK{
					log.Println("Bad StatusCode", resp.StatusCode)
					return fmt.Errorf("invalid tag: %s", params[0])
				}

				return nil
			}
		case modTagMember_t:
			return func(config FeedieConfig, params []string) error{
				tag_name := params[0]
				feeds := params[1:]
				changes := getModTagChanges(config, tag_name, feeds)
				
				for _, c := range changes{
					if c.add{
						resp, err := http.Get(fmt.Sprintf("%s%s/add_member?tag_name=%s&feed_url=%s",
						config.SERVER,config.PORT,url.QueryEscape(tag_name),url.QueryEscape(c.url))); if err != nil{
							log.Println(err)
							return err
						}
						defer resp.Body.Close()
						if resp.StatusCode != http.StatusOK {
							log.Println("Bad StatusCode", resp.StatusCode)
							return fmt.Errorf("invalid tag: %s", tag_name)
						}  
					} else{
						resp, err := http.Get(fmt.Sprintf("%s%s/del_member?tag_name=%s&feed_url=%s",
						config.SERVER,config.PORT,url.QueryEscape(tag_name),url.QueryEscape(c.url))); if err != nil{
							log.Println(err)
							return err
						}
						defer resp.Body.Close()
						if resp.StatusCode != http.StatusOK {
							log.Println("Bad StatusCode", resp.StatusCode)
							return fmt.Errorf("invalid tag: %s", tag_name)
						}  
					}
				}
				return nil
			}

			}
	
	return func(config FeedieConfig, params []string) error{
		_, _ = config, params
		return errors.New("invalid ActionType")
	}
}

type modTagMemberChange struct{
	add bool
	url string
}


func getModTagChanges(config FeedieConfig, tag string, newMembers []string) []modTagMemberChange{
	changes := []modTagMemberChange{}
	curMembers := getModTagOptions(config, tag)
	for _, cm := range curMembers{
		if cm.pu_selected{
			if !in(cm.Url, newMembers){
				changes = append(changes, modTagMemberChange{add: false, url: cm.Url})
			}
		} else{
			if in(cm.Url, newMembers){
				changes = append(changes, modTagMemberChange{add: true, url: cm.Url})
			}
		}
	}


	return changes
}


func getSelectOptions(config FeedieConfig) []list_source {
	ret := []list_source{}
	ret = append(ret, list_source{SrcType: Tag, SrcFunc: getAllFeedEntries, Title_field: "All feeds"})
	resp, err := http.Get(fmt.Sprintf("%s%s/get_tags",config.SERVER,config.PORT))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}
	var piece []list_source
	if err := json.NewDecoder(resp.Body).Decode(&piece); err != nil {
		log.Fatal(err)
	}
	for _, p := range piece{
		p.SrcType = Tag
		p.SrcFunc = getSrcFunc(p.SrcType, p.Title_field)
		//used for prefetching key
		p.Url = fmt.Sprintf("%s%s/get_entries?method=by_tag&value=%s",
		config.SERVER, config.PORT, url.QueryEscape(p.Title_field))
		ret = append(ret, p)
	}
	resp, err = http.Get(fmt.Sprintf("%s%s/get_feeds?method=all",config.SERVER,config.PORT))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret
	}
	if err := json.NewDecoder(resp.Body).Decode(&piece); err != nil {
		log.Fatal(err)
	}
	for _, p := range piece{
		p.SrcType = Feed
		p.SrcFunc = getSrcFunc(p.SrcType, p.Title_field)
		ret = append(ret, p)
	}
	return ret
}
func getFeedsByTag(config FeedieConfig, tag string) []string{
	type respFeed struct{
		Title string
		Url string
	}
	respFeeds := []respFeed{}
	ret := []string{}
	resp, err := http.Get(fmt.Sprintf("%s%s/get_feeds?method=by_tag&tag_name=%s",
	config.SERVER,config.PORT,url.QueryEscape(tag)))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}
	if err := json.NewDecoder(resp.Body).Decode(&respFeeds); err != nil {
		log.Fatal(err)
	}
	for _, f := range respFeeds{
		ret = append(ret, f.Url)
	}
	return ret
}

func getModTagOptions(config FeedieConfig, tag string) []popUpListItem{
	ret := []popUpListItem{}
	resp, err := http.Get(fmt.Sprintf("%s%s/get_feeds?method=by_tag&tag_name=%s",
	config.SERVER,config.PORT,url.QueryEscape(tag)))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}
	var piece []popUpListItem
	if err := json.NewDecoder(resp.Body).Decode(&piece); err != nil {
		log.Fatal(err)
	}
	for _, p := range piece{
		p.pu_selected = true
		ret = append(ret, p)
	}

	resp, err = http.Get(fmt.Sprintf("%s%s/get_feeds?method=by_tag&tag_name=%s&inverted=true",
	config.SERVER,config.PORT,url.QueryEscape(tag)))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}
	if err := json.NewDecoder(resp.Body).Decode(&piece); err != nil {
		log.Fatal(err)
	}
	for _, p := range piece{
		p.pu_selected = false
		ret = append(ret, p)
	}

	return ret
}
