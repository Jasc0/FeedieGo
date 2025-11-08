package main

import (
	"fmt"
	"strings"

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

	m.list.SetShowTitle(false)
	m.list.AdditionalFullHelpKeys = getSelectKeys(c)

	var sources []list.Item
	for _, src := range m.optFunc(m.config){
		sources = append(sources, src)
	}
	
	m.list.SetItems(sources)


	return m
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
	m.config.getFocusedStyle().Render(m.list.View()))
	
}
func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.ready{ m.ready = true; return  m, tea.WindowSize()}
	switch msg := msg.(type) {
	case RefreshMsg:
		c :=  m.Refresh(msg)
		return m, c
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height= msg.Height
		m.list.SetSize(msg.Width-2, msg.Height-2)
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		k := msg.String()
		if in(k,m.config.Keys["open"]){
			return initialEntriesModel(m.getSelectedSource().SrcFunc, m.config, m), tea.WindowSize()
		}
		if in(k,m.config.Keys["addFeed"]){
			if m.list.FilterState() != list.Filtering{
				return initialTextPopupModel(m.config, getActionFunc(addFeed_t), m,
				"Enter feed url:"), tea.WindowSize()
			}
		}
		if in(k,m.config.Keys["addTag"]){
			if m.list.FilterState() != list.Filtering{
				return initialTextPopupModel(m.config, getActionFunc(addTag_t), m,
				"Enter tag name:"), tea.WindowSize()
			}
		}
		if in(k,m.config.Keys["modTag"]){
			if m.list.FilterState() != list.Filtering{
				sel := m.list.SelectedItem()
				s, ok := sel.(list_source)
				if ok && s.SrcType == Tag{
					return initialListPopupModel(m.config, getActionFunc(modTagMember_t), getModTagOptions, true, m,
					"Select member feeds:", []string{s.Title_field}), tea.WindowSize()
				}
			}
		}
		if in(k,m.config.Keys["delete"]){
			if m.list.FilterState() != list.Filtering{
				cur := m.getSelectedSource()
				if cur.SrcType == Feed{
					return initialConfirmPopupModel(m.config, getActionFunc(delFeed_t), m,
					fmt.Sprintf("Delete feed %s ?", cur.Title_field),[]string{cur.Url}), tea.WindowSize()
				}
				if cur.SrcType == Tag{
					return initialConfirmPopupModel(m.config, getActionFunc(delTag_t), m,
					fmt.Sprintf("Delete tag %s ?", cur.Title_field),[]string{cur.Title_field}), tea.WindowSize()
				}
			}
		}
	}
	nl, c := m.list.Update(msg)
	m.list = nl
	return m, c
}

func (m *selectModel) Refresh (msg RefreshMsg) tea.Cmd{
	var sources []list.Item
	index := m.list.Index()
	for i, src := range m.optFunc(m.config){
		if src.Title_field == msg.itemName{
			index = i
		}
		sources = append(sources, src)
	}


	m.list.Select(index)
	setc := m.list.SetItems(sources)
	return tea.Batch(setc, tea.WindowSize()) 

}
func getSelectKeys(fc FeedieConfig) func() []key.Binding{
	selectCommands := []string{"addFeed", "addTag", "delete", "modTag"}	
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
