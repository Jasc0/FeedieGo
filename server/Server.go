package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func FeedieStartServer(port int){
	http.HandleFunc("/get_entries", getEntriesHandler)
	http.HandleFunc("/get_feeds", getFeedsHandler)
	http.HandleFunc("/get_tags", getTagsHandler)
	http.HandleFunc("/add_feed", addFeedHandler)
	http.HandleFunc("/del_feed", delFeedHandler)
	http.HandleFunc("/add_tag", addTagHandler)
	http.HandleFunc("/del_tag", delTagHandler)
	http.HandleFunc("/clear_members", clearTagHandler)
	http.HandleFunc("/add_member", AddTagMemberHandler)
	fmt.Printf("listening on :%d", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d",port), nil)
	if err != nil{
		log.Fatal(err)
	}
}

func getEntriesHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	order := timeOrder(DESC)
	method := r.URL.Query().Get("method")
	value := r.URL.Query().Get("value")
	if r.URL.Query().Has("rev"){ order = ASC}

	var data []FeedieEntry
	switch(method){
	case "all":
		log.Printf("serving /get_entries all feeds\n")
		data = DBGetAllTimeOrdered(order)

	case "by_tag":
		if value == ""{
			log.Printf("error serving /get_entries tag value empty")
			http.Error(w, "invalid tag name", http.StatusBadRequest)
			return
		}
		log.Printf("serving /get_entries tag=%s\n", value)
		data = DBGetByTagTimeOrdered(value, order)



	case "by_feed":
		if value == ""{
			log.Printf("error serving /get_entries feed value empty")
			http.Error(w, "invalid feed name", http.StatusBadRequest)
			return
		}
		log.Printf("serving /get_entries feed=%s\n", value)
		data = DBGetByFeedTimeOrdered(DBGetFeedByName(value), order)

	}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}



}

func getFeedsHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var data []FeedieFeed
	method :=r.URL.Query().Get("method")
	with_entries := false
	if r.URL.Query().Has("with_entries"){with_entries = true}
	switch(method){
	case "all":
		log.Printf("serving /get_feeds for all feeds\n")
		data = DBGetFeeds(with_entries)
	case "by_tag":
		tag_name := r.URL.Query().Get("tag_name")
		inverted := r.URL.Query().Has("inverted")
		log.Printf("serving /get_feeds for tag=%s\n", tag_name)
		data = DBGetFeedsByTag(tag_name, inverted)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}

func getTagsHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("serving /get_tags\n")
	data := DBGetTags()
	type src_object struct{
		Title string `json:"Title"`
	}
	objectified := []src_object{}
	for _, d := range data{
		objectified = append(objectified, src_object{Title: d})
		
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(objectified); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
}
func addFeedHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	url := r.URL.Query().Get("feed_url")
	if url == ""{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("error serving /add_feed url value empty")
		return
	}

	log.Printf("serving /add_feed, url=%s\n", url)
   addFeed(url)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	
}
func addFeed(url string){
	feed := parser(url)
	DBAddFeedWithEntries(*feed)
}

func delFeedHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	url := r.URL.Query().Get("feed_url")
	if url == ""{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("error serving /del_feed url value empty")
		return
	}

	log.Printf("serving /del_feed, url=%s\n", url)
   DBDelFeed(url)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func addTagHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("tag_name")
	if name == ""{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("error serving /add_tag tag_name value empty")
		return
	}

	log.Printf("serving /add_tag, tag_name=%s\n", name)
	DBAddTag(name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func delTagHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("tag_name")
	if name == ""{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("error serving /del_tag tag_name value empty")
		return
	}

	log.Printf("serving /del_tag, tag_name=%s\n", name)
	DBDelTag(name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func clearTagHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("tag_name")
	if name == ""{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("error serving /clear_tag tag_name value empty")
		return
	}

	log.Printf("serving /clear_tag, tag_name=%s\n", name)
	DBClearMembersTag(name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func AddTagMemberHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet{
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tag_name := r.URL.Query().Get("tag_name")
	feed_url := r.URL.Query().Get("feed_url")
	if tag_name == "" || feed_url == ""{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if tag_name == ""{log.Printf("error serving /add_member tag_name value empty") }
		if feed_url == ""{log.Printf("error serving /add_member feed_url value empty") }
		return
	}

	log.Printf("serving /add_member, tag_name=%s feed_url=%s\n", tag_name, feed_url)
	DBAddMembership(tag_name, feed_url)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
