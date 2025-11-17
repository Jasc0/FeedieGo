package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

const DEFAULT_PORT = 2550
const DEFAULT_REFRESH = 9000 // ~2.5 hrs


type FeedieServer struct{
	port int 
	timeOfNextRefresh int64
	refreshRate int64
	dbFilePath string
}

var feedieServer *FeedieServer
func feedieInit(){
	feedieServer = &FeedieServer{}
	if v, exists := os.LookupEnv("FEEDIE_SERVER_PORT"); exists{
		port, err:= strconv.Atoi(v)
		if err != nil{
			log.Fatal(err)
		}
		feedieServer.port = port
	}else{
		feedieServer.port = DEFAULT_PORT
	}
	if v, exists := os.LookupEnv("FEEDIE_SERVER_REFRESH_RATE"); exists{
		rr, err:= strconv.Atoi(v)
		if err != nil{
			log.Fatal(err)
		}
		feedieServer.refreshRate = int64(rr)
	}else{
		feedieServer.refreshRate = DEFAULT_REFRESH
	}
	feedieServer.timeOfNextRefresh += feedieServer.refreshRate
	if v, exists := os.LookupEnv("FEEDIE_SERVER_DB_PATH"); exists{
		path := v
		feedieServer.dbFilePath = path
	}else{
		if v, exists := os.LookupEnv("HOME"); exists{
			path := v
			feedieServer.dbFilePath = path + "/.local/share/feedie/feedie.db"
		}


	}

}

func main(){
	feedieInit()
	DBInit(feedieServer.dbFilePath)
	go refreshThread(feedieServer.refreshRate)
	FeedieStartServer(feedieServer.port)
}

func refreshThread(timeInSeconds int64){
	for true{
	feeds := DBGetFeeds(false)
	for _, feed := range feeds{
		go func (f FeedieFeed) {
		newFeed := parser(f.Url)
		if newFeed == nil{
			log.Printf("unable to refresh feed: %s", f.Url)
			return
		}
		DBAddFeedWithEntries(*newFeed)
		log.Printf("Refreshed feed: %s", newFeed.Url)
	}(feed)

	}
	for time.Now().Unix() < feedieServer.timeOfNextRefresh{
		time.Sleep(time.Duration(5) * time.Second)
	}
	log.Println("Refreshing feeds")
	feedieServer.timeOfNextRefresh += timeInSeconds
	}
}


