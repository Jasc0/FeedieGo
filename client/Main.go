package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)
const DEFAULT_H = 30
const DEFAULT_W = 30

type FeedieClientAction int
const(
	main_graphical FeedieClientAction = iota
	add_feed
)

var configPath string

func parseArgs() (FeedieClientAction, *string){
	if home, exists := os.LookupEnv("HOME"); exists{
		configPath = fmt.Sprintf("%s/.config/feedie/conf.json", home)

	} else{
		log.Fatal("You don't have a set $HOME variable? What the hell are you running?")
	}
	for i, arg := range os.Args{
		switch(arg){
		case "-c", "--config":
			if i+1 >= len(os.Args) {log.Fatal(errors.New("Expected value for --config"))}
			configPath = os.Args[i+1]
			continue
		case "--add_feed":
			if i+1 >= len(os.Args) {log.Fatal(errors.New("Expected value for --add_feed"))}
			return add_feed, &os.Args[i+1]
		}
	}
	return main_graphical, nil
}

func main() {
	action, feed := parseArgs()
	config := parseConfigFile(configPath)
	switch (action){
	case main_graphical:
		_ = feed
		p := tea.NewProgram(initialSelectModel(getSelectOptions, config), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	case add_feed:
		getActionFunc(addFeed_t)(config, []string{*feed})
	}
}
