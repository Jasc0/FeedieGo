package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func parseConfigFile(path string) FeedieConfig{
	 if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
        log.Fatal(err)
    }
	 if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		 writeDefaultConf(path)
    }
	 file, err := os.Open(path)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    var fc FeedieConfig
	 fc = getDefaultConf()
    if err := json.NewDecoder(file).Decode(&fc); err != nil {
        log.Fatal(err)
    }
	file.Close()
	 //write missing fields back to conf file
	 wfile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	 if err != nil {
		 log.Fatal(err)
	 }
	 defer wfile.Close()

	 enc := json.NewEncoder(wfile)
	 enc.SetIndent("", "  ")
	 err = enc.Encode(fc)
	 if err != nil{
		 log.Fatal(err)
	 }

	 if fc.getThumbnailBackend() == none{
		fc.ThumbnailRatio = 0
	 }

	 return fc

 }
 func writeDefaultConf(path string){

	 file, err := os.Create(path)
	 if err != nil {
		 log.Fatal(err)
	 }
	 defer file.Close()

	 enc := json.NewEncoder(file)
	 enc.SetIndent("", "  ")
	 err = enc.Encode(getDefaultConf())
	 if err != nil{
		 log.Fatal(err)
	 }
 }

 type FeedieConfig struct{
	 SERVER string `json:"server"` 
	 PORT string`json:"port"` 
	 BorderType string`json:"bordertype"` 
	 FocusFG string`json:"focusfg"` 
	 FocusBG string`json:"focusbg"` 
	 FocusBorderC string`json:"focusborderc"` 
	 NormalBG string`json:"normalbg"` 
	 NormalFG string`json:"normalfg"` 
	 NormalBorderC string`json:"normalborderc"` 
	 SelectBG string`json:"selectbg"` 
	 SelectFG string`json:"selectfg"` 
	 SelectCursor string`json:"selectcursor"` 
	 ThumbnailRatio float64 `json:"thumbnailratio"`
	 ThumbnailPath string`json:"thumbnailpath"` 
	 ThumbnailBackend string`json:"thumbnailbackend"` 
	 ThumbnailScaler string`json:"thumbnailscaler"` 
	 LinkCopyCommand string`json:"linkcopycommand"` 
	 TypeOpener map[string]string`json:"typeopener"` 
	 URLOpener map[string]string`json:"urlopener"` 
	 DefaultOpener string`json:"defaultopener"` 
	 Keys map[string][]string`json:"keys"` 

 }

 func (fc FeedieConfig) getThumbnailBackend() FeedieImageBackendProvider{
	 switch (fc.ThumbnailBackend){
	 case "kitty":
		 return kitty
	 case "ueberzug":
		 return ueberzug
	 default:
		 return none
	 }

 }

 func (fc FeedieConfig) getNormalStyle() lipgloss.Style {
	 var border lipgloss.Border
	 switch (fc.BorderType){
	 case "square":
		 border = lipgloss.NormalBorder()
	 case "rounded":
		 border = lipgloss.RoundedBorder()
	 }

	 return lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.NormalFG)).
	 Background(lipgloss.Color(fc.NormalBG)).
	 Border(border).
	 BorderForeground(lipgloss.Color(fc.NormalBorderC))
 }

 func (fc FeedieConfig) getFocusedStyle() lipgloss.Style {
	 var border lipgloss.Border
	 switch (fc.BorderType){
	 case "square":
		 border = lipgloss.NormalBorder()
	 case "rounded":
		 border = lipgloss.RoundedBorder()
	 default:
		 border = lipgloss.NormalBorder()
	 }

	 return lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.FocusFG)).
	 Background(lipgloss.Color(fc.FocusBG)).
	 Border(border).
	 BorderForeground(lipgloss.Color(fc.FocusBorderC))
 }
 func (fc FeedieConfig) getSelectedStyle() lipgloss.Style {

	 s := lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.SelectFG)).
	 Background(lipgloss.Color(fc.SelectBG))
	 s.BorderForeground(lipgloss.Color(fc.SelectCursor))
	 return s
 }

 func (fc FeedieConfig) getSelectDelegate() list.ItemDelegate{
	 del := FeedieSelectDelegate{config: fc}

	 del.Styles.SelectedTitle = fc.getSelectedStyle().Bold(true)
	 del.Styles.SelectedDesc = fc.getSelectedStyle()

	 del.Styles.NormalTitle = lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.NormalFG))
	 del.Styles.NormalDesc = lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.NormalFG)).Faint(true)


	 return del
 }
 func (fc FeedieConfig) getEntryDelegate() list.ItemDelegate{
	 del := FeedieEntryDelegate{config: fc}

	 del.Styles.SelectedTitle = fc.getSelectedStyle().Bold(true)
	 del.Styles.SelectedDesc = fc.getSelectedStyle()

	 del.Styles.NormalTitle = lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.NormalFG))
	 del.Styles.NormalDesc = lipgloss.NewStyle().
	 Foreground(lipgloss.Color(fc.NormalFG)).Faint(true)


	 return del
 }

 func (fc FeedieConfig) getYanker(pfc FeedieConfig, values []string){
	 link := values[0]
	 cmdParts := strings.Split(pfc.LinkCopyCommand," ")
	 cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	 stdin, err := cmd.StdinPipe()
    if err != nil {
        log.Fatal(err)
    }
	 if err := cmd.Start(); err != nil { log.Fatal(err) }
	 if _, err := io.WriteString(stdin, link); err != nil { log.Fatal(err) }
	 if err := stdin.Close(); err != nil { log.Fatal(err) } 
 }
 func (fc FeedieConfig) getLinkOpener(pfc FeedieConfig, values []string) error{
	 if len(values) < 2{
		 log.Fatal("odd length of values")
	 }	
	 link := FeedieLink{URL: values[0], Type: values[1]}
	 // match by URL first
	 // key = regex for url to match
	 // value = program to run
	 for key, value := range pfc.URLOpener {
		 re, err := regexp.Compile(key)
		 if err != nil {
			 return err
		 }
		 if re.MatchString(link.URL){
			 cmd := exec.Command(value, link.URL)
			 cmd.Stdin = nil
			 go cmd.Run()
			 return nil
		 }
	 }
	 // match by type second
	 // key = type of url i.e. "text/html"
	 // value = program to run
	 for key, value := range pfc.TypeOpener {
		 if link.Type == key{
			 cmd := exec.Command(value, link.URL)
			 cmd.Stdin = nil
			 go cmd.Run()
			 return nil
		 }
	 }

	 cmd := exec.Command(pfc.DefaultOpener, link.URL)
	 cmd.Stdin = nil
	 go cmd.Run()
	 return nil

 }



 func getDefaultConf() FeedieConfig{
	 fc :=  FeedieConfig{
		 SERVER: "http://localhost",
		 PORT: ":2550",
		 FocusBG: "#000000",
		 FocusFG: "#f9e0a1",
		 FocusBorderC: "#00ff00",
		 NormalBG: "#000000",
		 NormalFG: "#f9e0a1",
		 NormalBorderC: "#326416",
		 SelectBG: "#000000",
		 SelectFG: "#f9e0a1",
		 SelectCursor: "#00ff00",
		 BorderType: "square",
		 ThumbnailRatio: 0.4,
		 ThumbnailPath: "/tmp/feedie-go",
		 ThumbnailBackend: "kitty",
		 ThumbnailScaler: "fit_contain",
		 LinkCopyCommand: "xclip -i -selection clipboard",
		 DefaultOpener: "xdg-open",
		 URLOpener: make(map[string]string),
		 TypeOpener: make(map[string]string),
		 Keys: map[string][]string{
			 "open":{"enter"}, 
			 "copyLink":{"y"},
			 "addTag":{"t"}, 
			 "modTag":{"T"}, 
			 "addFeed":{"a"}, 
			 "delete":{"d"}, 
			 "quit":{"Q"},
			 "changeFocus":{"tab"},
			 "cursorDown":{"j","down"},
			 "cursorUp":{"k","up"},
			 "goToEnd":{"G"},
			 "goToStart":{"g"},
			 "filter":{"/"}, 
			 "feedMenu":{"m"},
			 "openMenu":{"o"}, 
			 "refresh":{"r"},
			 "help":{"?"},
			 "select":{" "},
		 },
	 }
	 return fc
 }
