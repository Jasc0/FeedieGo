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
		data = DBGetAllTimeOrdered(order)

	case "by_tag":
		if value == ""{
			http.Error(w, "invalid tag name", http.StatusBadRequest)
			return
		}
		data = DBGetByTagTimeOrdered(value, order)



	case "by_feed":
		if value == ""{
			http.Error(w, "invalid tag name", http.StatusBadRequest)
			return
		}
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
		data = DBGetFeeds(with_entries)
	case "by_tag":
		tag_name := r.URL.Query().Get("tag_name")
		inverted := r.URL.Query().Has("inverted")
		data = DBGetFeedsByTag(tag_name, inverted)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getTagsHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := DBGetTags()
	type parity_object struct{
		Title string `json:"Title"`
		//SrcType string `json:"SrcType"`
		//Url string `json:"Url"`
	}
	objectified := []parity_object{}
	for _, d := range data{
		objectified = append(objectified, parity_object{Title: d})
		
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(objectified); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		return
	}

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
		return
	}

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
		return
	}

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
		return
	}

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
		return
	}

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
		return
	}

	DBAddMembership(tag_name, feed_url)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
