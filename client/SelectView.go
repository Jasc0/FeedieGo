package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectModel struct {
	optFunc func(FeedieConfig) []list_source
	config  FeedieConfig
	ready   bool
	list    list.Model
	height  int
	width   int
	popup   popUpModel
}

func (m selectModel) getSelectedSource() list_source {
	if ls, ok := m.list.SelectedItem().(list_source); ok {
		return ls
	}
	return list_source{}
}

var preloadMu sync.Mutex
var preloadMap map[string][]list_entry

func initialSelectModel(optFunc func(FeedieConfig) []list_source, config FeedieConfig) selectModel {
	m := selectModel{
		config:  config,
		optFunc: optFunc,
		width:   defaultW,
		height:  defaultH,
		ready:   false,
		list:    list.New([]list.Item{}, config.getSelectDelegate(), defaultW, defaultH),
	}

	keyMap := list.KeyMap{
		CursorUp: key.NewBinding(
			key.WithKeys(config.Keys["cursorUp"]...),
			key.WithHelp(strings.Join(config.Keys["cursorUp"], "\\"), "Up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys(config.Keys["cursorDown"]...),
			key.WithHelp(strings.Join(config.Keys["cursorDown"], "\\"), "Down"),
		),
		Filter: key.NewBinding(
			key.WithKeys(config.Keys["filter"]...),
			key.WithHelp(strings.Join(config.Keys["filter"], "\\"), "Filter"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys(config.Keys["goToEnd"]...),
			key.WithHelp(strings.Join(config.Keys["goToEnd"], "\\"), "End"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys(config.Keys["goToStart"]...),
			key.WithHelp(strings.Join(config.Keys["goToStart"], "\\"), "Start"),
		),
		Quit: key.NewBinding(
			key.WithKeys(config.Keys["quit"]...),
			key.WithHelp(strings.Join(config.Keys["quit"], "\\"), "Quit"),
		),
		CancelWhileFiltering: key.NewBinding(
			key.WithKeys(tea.KeyEsc.String()),
			key.WithHelp(tea.KeyEsc.String(), "Cancel filtering"),
		),
		AcceptWhileFiltering: key.NewBinding(
			key.WithKeys(tea.KeyEnter.String()),
			key.WithHelp(tea.KeyEnter.String(), "Cancel filtering"),
		),
		ShowFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "Show help"),
		),
	}

	m.list.KeyMap = keyMap
	m.list.SetShowTitle(false)
	m.list.AdditionalFullHelpKeys = getSelectKeys(config)
	m.list.Help.ShowAll = true
	m.list.SetShowHelp(false)

	var sources []list.Item
	for _, src := range m.optFunc(m.config) {
		sources = append(sources, src)
	}

	m.list.SetItems(sources)
	preloadMap = make(map[string][]list_entry)
	m.preloadFeeds(preloadAmt)

	return m
}

func (m selectModel) preloadFeeds(preload int) {
	var items []list.Item
	if m.list.FilterState() == list.FilterApplied {
		items = m.list.VisibleItems()
	} else {
		items = m.list.Items()
	}

	startIdx := max(0, m.list.Index()-preload/2)
	endIdx := min(startIdx+preload, len(items))
	srcs := []list_source{}
	for _, item := range items[startIdx:endIdx] {
		src := item.(list_source)
		srcs = append(srcs, src)
	}

	for _, src := range srcs {
		preloadMu.Lock()
		_, ok := preloadMap[src.Url]
		preloadMu.Unlock()
		if ok {
			continue
		}
		go func(s list_source) {
			entries := s.SrcFunc(m.config, 0)
			preloadMu.Lock()
			preloadMap[s.Url] = entries
			preloadMu.Unlock()
		}(src)
	}
}

func (m selectModel) getPreloaded(url string) []list_entry {
	preloadMu.Lock()
	v, ok := preloadMap[url]
	preloadMu.Unlock()
	if ok {
		return v
	}
	return []list_entry{}
}

func (m selectModel) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m selectModel) View() string {
	if !m.ready {
		return "loading... \n"
	}

	return lipgloss.Place(m.width, m.height,
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
		switch msg.MsgType {
		case addTagMsg:
			cmd := m.Refresh(msg)
			sel := m.list.SelectedItem()
			s, ok := sel.(list_source)
			if !ok {
				log.Fatal("ERR Modifying tags")
			}
			return initialListPopupModel(m.config, getActionFunc(modTagMember_t), getModTagOptions, true, m,
				"Select member feeds:", []string{s.Title_field}, RefreshCmd), cmd
		case refreshMsg:
			cmd := m.Refresh(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-2, msg.Height-2)
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		if m.list.SettingFilter() {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		}

		k := msg.String()
		if in(k, m.config.Keys["cursorUp"]) {
			m.list.CursorUp()
			m.preloadFeeds(preloadAmt)
			return m, nil
		}

		if in(k, m.config.Keys["cursorDown"]) {
			m.list.CursorDown()
			m.preloadFeeds(preloadAmt)
			return m, nil
		}

		if in(k, m.config.Keys["open"]) {
			selected := m.getSelectedSource()
			return initialEntriesModel(selected.SrcFunc, m.config, m,
				m.getPreloaded(selected.Url)),
				tea.WindowSize()
		}

		if in(k, m.config.Keys["addFeed"]) {
			return initialTextPopupModel(m.config, getActionFunc(addFeed_t), m,
				"Enter feed url:", RefreshCmd), tea.WindowSize()
		}

		if in(k, m.config.Keys["addTag"]) {
			return initialTextPopupModel(m.config, getActionFunc(addTag_t), m,
				"Enter tag name:", addTagCmd), tea.WindowSize()
		}

		if in(k, m.config.Keys["modTag"]) {
			sel := m.list.SelectedItem()
			s, ok := sel.(list_source)
			if ok && s.SrcType == Tag {
				return initialListPopupModel(m.config, getActionFunc(modTagMember_t), getModTagOptions, true, m,
					"Select member feeds:", []string{s.Title_field}, RefreshCmd), tea.WindowSize()
			}
		}

		if in(k, m.config.Keys["delete"]) {
			selected := m.getSelectedSource()
			if selected.SrcType == Feed {
				return initialConfirmPopupModel(m.config, getActionFunc(delFeed_t), m,
					fmt.Sprintf("Delete feed %s ?", selected.Title_field), []string{selected.Url}, RefreshCmd), tea.WindowSize()
			}
			if selected.SrcType == Tag {
				return initialConfirmPopupModel(m.config, getActionFunc(delTag_t), m,
					fmt.Sprintf("Delete tag %s ?", selected.Title_field), []string{selected.Title_field}, RefreshCmd), tea.WindowSize()
			}
		}

		if in(k, m.config.Keys["refresh"]) {
			m.clearPreload()
			return m, RefreshCmd("")
		}

		if in(k, m.config.Keys["help"]) {
			if m.list.ShowHelp() {
				m.list.SetShowHelp(false)
			} else {
				m.list.SetShowHelp(true)
			}
			return m, nil
		}

		if in(k, m.config.Keys["filter"]) {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		}

		if in(k, m.config.Keys["quit"]) {
			return m, tea.Quit
		}

		newList, cmd := m.list.Update(msg)
		m.list = newList
		return m, cmd

	default:
		newList, cmd := m.list.Update(msg)
		m.list = newList
		return m, cmd
	}
	return m, nil
}

func (m *selectModel) Refresh(msg FeedieMsg) tea.Cmd {
	var sources []list.Item
	index := m.list.Index()
	prevFilter := ""
	if m.list.IsFiltered() {
		m.list.SetFilterState(list.Unfiltered)
		prevFilter = m.list.FilterValue()
	}
	for i, src := range m.optFunc(m.config) {
		if src.Title_field == msg.Item {
			index = i
		}
		sources = append(sources, src)
	}

	cmd := m.list.SetItems(sources)
	if prevFilter != "" {
		m.list.SetFilterText(prevFilter)
	}
	m.list.Select(index)

	return tea.Batch(cmd, tea.WindowSize())
}

func (m *selectModel) clearPreload() {
	preloadMu.Lock()
	for k := range preloadMap {
		delete(preloadMap, k)
	}
	preloadMu.Unlock()
}

func getSelectKeys(config FeedieConfig) func() []key.Binding {
	selectCommands := []string{"addFeed", "addTag", "delete", "modTag", "refresh"}
	ret := []key.Binding{}
	for command, keys := range config.Keys {
		if in(command, selectCommands) {
			binding := key.NewBinding(
				key.WithKeys(keys...),
				key.WithHelp(strings.Join(keys, "\\"), command),
			)
			ret = append(ret, binding)
		}
	}
	return func() []key.Binding { return ret }
}
