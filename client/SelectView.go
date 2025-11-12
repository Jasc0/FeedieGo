package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	_ "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


type selectModel struct{
	optFunc func(FeedieConfig)[]list_source
	config FeedieConfig
	ready bool
	list list.Model
	height int
	width int
	popup popUpModel
}

func (m selectModel) getSelectedSource() list_source{
	if ls, ok := m.list.SelectedItem().(list_source); ok{
		return ls
	}
	return list_source{}
}

var preloadMu sync.Mutex
var preloadMap map[string][]list_entry
func initialSelectModel( f func(FeedieConfig)[]list_source, c FeedieConfig) selectModel {
	
	m := selectModel{
		config: c,
		optFunc : f,
		width: DEFAULT_W,
		height: DEFAULT_H,
		ready: false,
		list: list.New([]list.Item{}, c.getSelectDelegate(),
		DEFAULT_W , DEFAULT_H),
		
	}

	kb := list.KeyMap{
		CursorUp:  key.NewBinding(key.WithKeys(c.Keys["cursorUp"]...),
		key.WithHelp(strings.Join(c.Keys["cursorUp"],"\\"), "Up")),

		CursorDown : key.NewBinding(key.WithKeys(c.Keys["cursorDown"]...),
		key.WithHelp(strings.Join(c.Keys["cursorDown"],"\\"), "Down")),

		Filter : key.NewBinding(key.WithKeys(c.Keys["filter"]...),
		key.WithHelp(strings.Join(c.Keys["filter"],"\\"), "Filter")),

		GoToEnd : key.NewBinding(key.WithKeys(c.Keys["goToEnd"]...),
		key.WithHelp(strings.Join(c.Keys["goToEnd"],"\\"), "End")),

		GoToStart : key.NewBinding(key.WithKeys(c.Keys["goToStart"]...),
		key.WithHelp(strings.Join(c.Keys["goToStart"],"\\"), "Start")),

		Quit : key.NewBinding(key.WithKeys(c.Keys["quit"]...),
		key.WithHelp(strings.Join(c.Keys["quit"],"\\"), "Quit")),

		CancelWhileFiltering : key.NewBinding(key.WithKeys(tea.KeyEsc.String()),
		key.WithHelp(tea.KeyEsc.String(), "Cancel filtering")),
		AcceptWhileFiltering : key.NewBinding(key.WithKeys(tea.KeyEnter.String()),
		key.WithHelp(tea.KeyEnter.String(), "Cancel filtering")),
		ShowFullHelp : key.NewBinding(key.WithKeys("?"),
		key.WithHelp("?", "Show help")),

	}

	m.list.KeyMap = kb
	m.list.SetShowTitle(false)
	m.list.AdditionalFullHelpKeys = getSelectKeys(c)
	m.list.Help.ShowAll = true
	m.list.SetShowHelp(false)

	var sources []list.Item
	for _, src := range m.optFunc(m.config){
		sources = append(sources, src)
	}
	
	m.list.SetItems(sources)
	preloadMap = make(map[string][]list_entry)
	m.preloadFeeds(PRELOAD_AMT)

	return m
}

func (m selectModel) preloadFeeds(preload int){
	var items []list.Item
	if m.list.FilterState() == list.FilterApplied{
		items = m.list.VisibleItems()
	}else{
		items = m.list.Items()
	}

	si := max(0,m.list.Index() - preload/2)
	ei := min(si+preload, len(items))
	srcs := []list_source{}
	for _, it := range items[si:ei]{
		it := it.(list_source)
		srcs = append(srcs, it)
	}

	for _, src := range srcs{
		preloadMu.Lock()
		_, ok := preloadMap[src.Url]
	preloadMu.Unlock()
		if ok{
			continue
		}
		if !ok{
			go func(s list_source){
				LEs := s.SrcFunc(m.config)
				preloadMu.Lock()
				preloadMap[s.Url] = LEs
				preloadMu.Unlock()
			}(src)
		}
	}

}

func (m selectModel) getPreloaded(url string)[]list_entry{
	preloadMu.Lock()
	v, ok := preloadMap[url]
	preloadMu.Unlock()
	if ok{
		return v
	}
	return []list_entry{}
}

func (m selectModel) Init() tea.Cmd {
	return tea.WindowSize()

}

func (m selectModel) View() string {
	if !m.ready{
		return "loading... \n"
	}

	return lipgloss.Place(m.width,m.height,
	lipgloss.Center,
	lipgloss.Center,
	m.config.getFocusedStyle().
	Width(m.width/2).
	Height(m.height-2).
	Render(m.list.View()))
	
}
func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FeedieMsg:
		switch msg.MsgType{
		case addTagMsg:
			c :=  m.Refresh(msg)
			sel := m.list.SelectedItem()
			s, ok := sel.(list_source)
			if !ok{
				log.Fatal("ERR Modifying tags")
			}
			return initialListPopupModel(m.config, getActionFunc(modTagMember_t), getModTagOptions, true, m,
			"Select member feeds:", []string{s.Title_field}, RefreshCmd), c
		case refreshMsg:
			c :=  m.Refresh(msg)
			return m, c
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height= msg.Height
		m.list.SetSize(msg.Width-2, msg.Height-2)
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		if m.list.SettingFilter(){
			nl, c := m.list.Update(msg)
			m.list = nl
			return m, c
		}
		k := msg.String()
		if in(k,m.config.Keys["cursorUp"]){
			m.list.CursorUp()
			return m, nil
		}
		if in(k,m.config.Keys["cursorDown"]){
			m.list.CursorDown()
			return m, nil
		}
		if in(k,m.config.Keys["open"]){
			cur := m.getSelectedSource()
			return initialEntriesModel(cur.SrcFunc, m.config, m,
			m.getPreloaded(cur.Url)), 
			tea.WindowSize()
		}

		if in(k,m.config.Keys["addFeed"]){
			return initialTextPopupModel(m.config, getActionFunc(addFeed_t), m,
			"Enter feed url:", RefreshCmd ), tea.WindowSize()
		} 
		if in(k,m.config.Keys["addTag"]){
			return initialTextPopupModel(m.config, getActionFunc(addTag_t), m,
			"Enter tag name:", addTagCmd), tea.WindowSize()
		}
		if in(k,m.config.Keys["modTag"]){
			sel := m.list.SelectedItem()
			s, ok := sel.(list_source)
			if ok && s.SrcType == Tag{
				return initialListPopupModel(m.config, getActionFunc(modTagMember_t), getModTagOptions, true, m,
				"Select member feeds:", []string{s.Title_field}, RefreshCmd), tea.WindowSize()
			}
		}
		if in(k,m.config.Keys["delete"]){
			cur := m.getSelectedSource()
			if cur.SrcType == Feed{
				return initialConfirmPopupModel(m.config, getActionFunc(delFeed_t), m,
				fmt.Sprintf("Delete feed %s ?", cur.Title_field),[]string{cur.Url},RefreshCmd), tea.WindowSize()
			}
			if cur.SrcType == Tag{
				return initialConfirmPopupModel(m.config, getActionFunc(delTag_t), m,
				fmt.Sprintf("Delete tag %s ?", cur.Title_field),[]string{cur.Title_field},RefreshCmd), tea.WindowSize()
			}
		}
		if in(k,m.config.Keys["refresh"]){
			return m, RefreshCmd("")
		}
		if in(k,m.config.Keys["help"]){
			if m.list.ShowHelp(){
				m.list.SetShowHelp(false)
			} else{
				m.list.SetShowHelp(true)
			}
			return m,nil
		}
		if in(k,m.config.Keys["filter"]){
			nl, c := m.list.Update(msg)
			m.list = nl
			return m, c
		}
		if in(k,m.config.Keys["quit"]){
			return m, tea.Quit
		}
	default:
		nl, c := m.list.Update(msg)
		m.list = nl
		return m, c
	}
	m.preloadFeeds(PRELOAD_AMT)
	return m, nil
}
func (m *selectModel) Refresh (msg FeedieMsg) tea.Cmd{
	var sources []list.Item
	index := m.list.Index()
	prevFilter := ""
	if m.list.IsFiltered(){
		m.list.SetFilterState(list.Unfiltered)
		prevFilter = m.list.FilterValue()
	}
	for i, src := range m.optFunc(m.config){
		if src.Title_field == msg.Item{
			index = i
		}
		sources = append(sources, src)
	}


	setc := m.list.SetItems(sources)
	if prevFilter != "" {m.list.SetFilterText(prevFilter)}
	m.list.Select(index)
	preloadMu.Lock()
	for k := range preloadMap {
		delete(preloadMap, k)
	}
	preloadMu.Unlock()
	return tea.Batch(setc, tea.WindowSize()) 

}
func getSelectKeys(fc FeedieConfig) func() []key.Binding{
	selectCommands := []string{"addFeed", "addTag", "delete", "modTag", "refesh"}	
	ret := []key.Binding{}

	for command, keys := range fc.Keys{
		if in(command, selectCommands){
			kb := key.NewBinding(
				key.WithKeys(keys...),
				key.WithHelp(strings.Join(keys,"\\"),command),
			)
			ret = append(ret, kb)
		}


	}
	return func () []key.Binding {return ret}
}
