package main

import (
	"fmt"
	"io"
	"slices"
	"time"

	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func in(k string, match_to[]string) bool{
	return slices.Contains(match_to, k)
}

type FMsgType int 
const(
	refreshMsg FMsgType = iota 
	addTagMsg
)

type FeedieMsg struct{MsgType FMsgType; Item any}
func FeedieCmd(t FMsgType,i any) tea.Cmd {
	return func () tea.Msg { return  FeedieMsg{MsgType: t, Item: i}}
}
func RefreshCmd(name string) tea.Cmd{
	return func () tea.Msg { return FeedieMsg{MsgType: refreshMsg, Item: name}}
}
func addTagCmd(name string) tea.Cmd{
	return FeedieCmd(addTagMsg, name)
}

type FeedieLink struct {
	URL string
	Type string
}
type list_entry struct{
	Title_field string `json:"Title"`
	Author string `json:"Author"`
	Thumbnail string `json:"Thumbnail"`
	Description_field string `json:"Description"`
	Published int `json:"Published"`
	Links []FeedieLink `json:"Links"`
}
func (i list_entry) Title() string       { return i.Title_field }
func (i list_entry) Description() string { return i.Author }
func (i list_entry) FilterValue() string { return i.Title_field }

func (i list_entry) FullDescription(Width int) string{
	base := ""
	base += fmt.Sprintf("%s\n %s\n",
		lipgloss.NewStyle().Bold(true).Render(i.Title_field),
	lipgloss.NewStyle().Faint(true).Render(i.published()))
	base+= strings.Repeat("-", Width)
	base+= "\n"
	md, err := htmltomarkdown.ConvertString(i.Description_field)
	if err != nil{
		md = i.Description_field
	}
	base += md

	return lipgloss.NewStyle().Width(Width).Render(base)
	
}

func (i list_entry) published () string{
	return time.Unix(int64(i.Published), 0).Format(time.RFC1123)
}
func (i list_entry) getLinks(fc FeedieConfig, unused string) []popUpListItem{
	//parameters ununsed, only there so I can use one getInitialListPopupModel
	_, _ = fc, unused
	ret := []popUpListItem{}
	for _, link := range i.Links{
		ret = append(ret, popUpListItem{Title_Field: link.URL, Url: link.Type})
	}
	
	return ret
}


type FeedieSelectDelegate struct{
	list.DefaultDelegate
	config FeedieConfig
}
type FeedieEntryDelegate struct{
	list.DefaultDelegate
	config FeedieConfig
}
func (d FeedieEntryDelegate) Height() int {return 2}
func (d FeedieEntryDelegate) Spacing() int {return 1}
func (d FeedieEntryDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(list_entry)
	if ok {
		bar := " " // remove the bar (or set to "▶ " or "→ ")
		title := i.Title()
		desc := i.Description()

		if index == m.Index() {
			bar = "┃" // custom indicator
			title = d.Styles.SelectedTitle.MaxWidth(m.Width()-2).MaxHeight(1).Render(title)
			bar = d.Styles.SelectedTitle.Foreground(lipgloss.Color(d.config.SelectCursor)).Render(bar)
			desc = d.Styles.SelectedDesc.MaxWidth(m.Width()-2).MaxHeight(1).Render(desc)
		}else{
			bar = " " // custom indicator
			title = d.Styles.NormalTitle.MaxWidth(m.Width() - 2).MaxHeight(1).Render(title)
			bar = d.Styles.NormalTitle.Foreground(lipgloss.Color(d.config.SelectCursor)).Render(bar)
			desc = d.Styles.NormalDesc.MaxWidth(m.Width()-2).MaxHeight(1).Render(desc)
		}

		fmt.Fprintf(w, "%s%s\n%s%s", bar, title,bar, desc)
	}

}
func (d FeedieSelectDelegate) Height() int {return 1}
func (d FeedieSelectDelegate) Spacing() int {return 1}
func (d FeedieSelectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(list_source)
	if ok {
		bar := " " // remove the bar (or set to "▶ " or "→ ")
		title := i.Title()
		desc := i.Description()

		if index == m.Index() {
			bar = "┃" // custom indicator
			title = d.Styles.SelectedTitle.MaxWidth(m.Width()/2-2).Render(title)
			bar = d.Styles.SelectedTitle.Foreground(lipgloss.Color(d.config.SelectCursor)).Render(bar)
			desc = d.Styles.SelectedDesc.Render(desc)
		}else{
			bar = " " // custom indicator
			title = d.Styles.NormalTitle.MaxWidth(m.Width()/2 - 2).Render(title)
			bar = d.Styles.NormalTitle.Foreground(lipgloss.Color(d.config.SelectCursor)).Render(bar)
			desc = d.Styles.NormalDesc.Render(desc)
		}

		fmt.Fprintf(w, "%s%s", bar, title)
	}

}

type SourceType int

const (
	Tag SourceType = iota
	Feed

)

type list_source struct{
	Title_field string `json:"Title"`
	SrcType SourceType `json:"SrcType"`
	SrcFunc func(FeedieConfig) []list_entry 
	Url string `json:"Url"`
}
func (i list_source) Title() string       { 
	var icon string
	switch(i.SrcType){
	case Tag:
		icon = "#"
	default:
		icon = ""
	}
	return fmt.Sprintf("%s%s",icon,i.Title_field) 
}
func (i list_source) Description() string { return "" }
func (i list_source) FilterValue() string { return i.Title_field }
