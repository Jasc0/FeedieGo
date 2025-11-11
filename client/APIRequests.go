package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func getAllFeedEntries(config FeedieConfig) []list_entry {
	ret := []list_entry{}

	resp, err := http.Get(fmt.Sprintf("%s%s/get_entries?method=all",config.SERVER,config.PORT))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}

	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.Fatal(err)
	}

	return ret
}

func getSrcFunc(SrcType SourceType, protokey string) func(FeedieConfig) []list_entry {
	key := url.QueryEscape(protokey)

	switch (SrcType){
	case Tag:
		return func (config FeedieConfig) []list_entry {
			ret := []list_entry{}

			resp, err := http.Get(fmt.Sprintf("%s%s/get_entries?method=by_tag&value=%s",config.SERVER,config.PORT,key))
			if err != nil{
				log.Println(err)
				return ret
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK{
				log.Println("Bad StatusCode")
				return ret
			}

			if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
				log.Fatal(err)
			}
			return ret
		}
	case Feed:
		return func (config FeedieConfig) []list_entry{
			ret := []list_entry{}
			resp, err := http.Get(fmt.Sprintf("%s%s/get_entries?method=by_feed&value=%s",config.SERVER,config.PORT,key)); if err != nil{
				log.Println(err)
				return ret
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK{
				log.Println("Bad StatusCode", resp.StatusCode)
				return ret
			}

			if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
				log.Fatal(err)
			}
			return ret
		}
	}
	return func( config FeedieConfig) []list_entry{
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
					return errors.New("invalid feed url: %s"+ params[0])
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
					return errors.New("invalid feed url: %s"+ params[0])
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
					return errors.New("invalid tag: %s"+ params[0])
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
					return errors.New("invalid tag: %s"+ params[0])
				}

				return nil
			}
		case modTagMember_t:
			return func(config FeedieConfig, params []string) error{
				tag_name := params[0]
				feeds := params[1:]
				resp, err := http.Get(fmt.Sprintf("%s%s/clear_members?tag_name=%s",
					config.SERVER,config.PORT,tag_name)); if err != nil{
					log.Println(err)
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK{
					log.Println("Bad StatusCode", resp.StatusCode)
					return errors.New("invalid tag: %s"+ params[0])
				}

				resp.Body.Close()

				for _, f := range feeds{

					resp, err := http.Get(fmt.Sprintf("%s%s/add_member?tag_name=%s&feed_url=%s",
					config.SERVER,config.PORT,tag_name,url.QueryEscape(f))); if err != nil{
						log.Println(err)
						return err
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK{
						log.Println("Bad StatusCode", resp.StatusCode)
						return errors.New("invalid tag: %s"+ params[0])
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
	var peice []list_source
	if err := json.NewDecoder(resp.Body).Decode(&peice); err != nil {
		log.Fatal(err)
	}
	for _, p := range peice{
		p.SrcType = Tag
		p.SrcFunc = getSrcFunc(p.SrcType, p.Title_field)
		//used for prefetching key
		p.Url = fmt.Sprintf("%s%s/get_entries?method=by_tag&value=%s",
		config.SERVER, config.PORT, p.Title_field)
		ret = append(ret, p)
	}
	resp.Body.Close()
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
	if err := json.NewDecoder(resp.Body).Decode(&peice); err != nil {
		log.Fatal(err)
	}
	for _, p := range peice{
		p.SrcType = Feed
		p.SrcFunc = getSrcFunc(p.SrcType, p.Title_field)
		ret = append(ret, p)
	}
	return ret
}

func getModTagOptions(config FeedieConfig, tag string) []popUpListItem{
	ret := []popUpListItem{}
	resp, err := http.Get(fmt.Sprintf("%s%s/get_feeds?method=by_tag&tag_name=%s",
	config.SERVER,config.PORT,tag))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}
	var peice []popUpListItem
	if err := json.NewDecoder(resp.Body).Decode(&peice); err != nil {
		log.Fatal(err)
	}
	for _, p := range peice{
		p.pu_selected = true
		ret = append(ret, p)
	}
	resp.Body.Close()

	resp, err = http.Get(fmt.Sprintf("%s%s/get_feeds?method=by_tag&tag_name=%s&inverted=true",
	config.SERVER,config.PORT,tag))
	if err != nil{
		log.Println(err)
		return ret
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		log.Println("Bad StatusCode")
		return ret

	}
	if err := json.NewDecoder(resp.Body).Decode(&peice); err != nil {
		log.Fatal(err)
	}
	for _, p := range peice{
		p.pu_selected = false
		ret = append(ret, p)
	}

	return ret
}
