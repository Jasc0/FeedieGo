package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type popUpType int

const (
	text_t popUpType=iota
	confirm_t
	list_t
)


type popUpModel struct{
	width, height int
	prevModel tea.Model
	display popUpType
	textinput textinput.Model
	confirm bool
	list list.Model
	listSrcFunc func(FeedieConfig, string) []popUpListItem
	listMultiSelect bool
	action func(FeedieConfig, []string) error
	config FeedieConfig
	prompt string
	values []string // used for passing data for action call
}
func initialTextPopupModel(fc FeedieConfig, action func(FeedieConfig, []string) error, prev tea.Model, prompt string) tea.Model{
	m := popUpModel{
		prevModel: prev,
		display: text_t,
		textinput: textinput.New(),
		action: action,
		config: fc,
		prompt: prompt,
	}
	m.textinput.Focus()
	return m
}
func initialConfirmPopupModel(fc FeedieConfig, action func(FeedieConfig, []string) error, prev tea.Model, prompt string, values []string) tea.Model{
	m := popUpModel{
		prevModel: prev,
		display: confirm_t,
		confirm: false,
		action: action,
		config: fc,
		prompt: prompt,
		values: values,
	}
	return m
}
func initialListPopupModel(fc FeedieConfig, action func(FeedieConfig, []string) error, srcFunc func(FeedieConfig, string) []popUpListItem, multi bool, prev tea.Model, prompt string, values []string) tea.Model{
	m := popUpModel{
		prevModel: prev,
		display: list_t,
		list: list.New([]list.Item{},FeediePopUpDelegate{multi: multi},DEFAULT_W,DEFAULT_H),
		listMultiSelect: multi,
		action: action,
		config: fc,
		prompt: prompt,
		values: values,
	}
	kb := list.KeyMap{
		CursorUp:  key.NewBinding(key.WithKeys(fc.Keys["cursorUp"]...),
		key.WithHelp(strings.Join(fc.Keys["cursorUp"],"\\"), "Up")),

		CursorDown : key.NewBinding(key.WithKeys(fc.Keys["cursorDown"]...),
		key.WithHelp(strings.Join(fc.Keys["cursorDown"],"\\"), "Down")),

		Filter : key.NewBinding(key.WithKeys(fc.Keys["filter"]...),
		key.WithHelp(strings.Join(fc.Keys["filter"],"\\"), "Filter")),

		GoToEnd : key.NewBinding(key.WithKeys(fc.Keys["goToEnd"]...),
		key.WithHelp(strings.Join(fc.Keys["goToEnd"],"\\"), "End")),

		GoToStart : key.NewBinding(key.WithKeys(fc.Keys["goToStart"]...),
		key.WithHelp(strings.Join(fc.Keys["goToStart"],"\\"), "Start")),

		Quit : key.NewBinding(key.WithKeys(fc.Keys["quit"]...),
		key.WithHelp(strings.Join(fc.Keys["quit"],"\\"), "Quit")),

		CancelWhileFiltering : key.NewBinding(key.WithKeys(tea.KeyEsc.String()),
		key.WithHelp(tea.KeyEsc.String(), "Cancel filtering")),
		AcceptWhileFiltering : key.NewBinding(key.WithKeys(tea.KeyEnter.String()),
		key.WithHelp(tea.KeyEnter.String(), "Cancel filtering")),
		ShowFullHelp : key.NewBinding(key.WithKeys("?"),
		key.WithHelp("?", "Show help")),

	}

	m.list.KeyMap = kb
	m.list.SetFilteringEnabled(true)
	m.list.SetShowTitle(false)
	
	var popUpLIs []popUpListItem
	if len(values) == 1{
		popUpLIs = srcFunc(fc, values[0])
	} else {
		popUpLIs = srcFunc(fc, "")
	}
	regLIs := make([]list.Item,len(popUpLIs))
	for i, p := range popUpLIs{
		regLIs[i] = p
	}
	m.list.SetItems([]list.Item(regLIs))

	return m
}

func (m popUpModel) Init() tea.Cmd {
	return nil
}

func (m popUpModel) View() string {
	var base string
	switch(m.display){
	case text_t:
		base = m.textinput.View()	
	case confirm_t:
		if m.confirm{
			base = lipgloss.JoinHorizontal(lipgloss.Center,
			m.config.getNormalStyle().Render("Cancel"),
			lipgloss.NewStyle().PaddingLeft(5).PaddingRight(5).Render(""),
			m.config.getFocusedStyle().Render("Confirm"))
		}else{
			base = lipgloss.JoinHorizontal(lipgloss.Center,
			m.config.getFocusedStyle().Render("Cancel"),
			lipgloss.NewStyle().PaddingLeft(5).PaddingRight(5).Render(""),
			m.config.getNormalStyle().Render("Confirm"))
		}
	case list_t:
		base = m.list.View()
	}
	base =  "\n" + base
	return lipgloss.Place(m.width,m.height,
		lipgloss.Center,lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
		m.config.getNormalStyle().Padding(5).Render(lipgloss.JoinVertical(lipgloss.Center,m.prompt, base)),
	   lipgloss.NewStyle().Faint(true).Render("Press Escape to exit/cancel")))
}


func (m popUpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.display {

	case text_t:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height= msg.Height
			m.textinput.Width= m.width /2
		case tea.KeyMsg:
			switch(msg.Type){
			case tea.KeyEnter:
				value := strings.TrimSpace(m.textinput.Value())
				m.values = []string{value}
				err := m.action(m.config, m.values)
				if err != nil{
					log.Fatal(err)
				}

				return m.prevModel, RefreshCmd(value)

			case tea.KeyEsc :
				return m.prevModel, tea.WindowSize()
			}

		}
		nm, c := m.textinput.Update(msg)
		m.textinput = nm
		return m, c


	case confirm_t:
		switch msg := msg.(type){
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height= msg.Height
		case tea.KeyMsg:
			switch msg.Type{
			case tea.KeyEnter:
				if m.confirm{
					err := m.action(m.config, m.values)
					if err != nil{
						log.Fatal(err)
					}

				}
				return m.prevModel, RefreshCmd("")
			case tea.KeyTab, tea.KeyRight, tea.KeyLeft:
				m.confirm = !m.confirm
				return m, nil
			case tea.KeyEsc:
				return m.prevModel, RefreshCmd("")
			}
		}
	case list_t:
		switch msg := msg.(type){
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height= msg.Height
			m.list.SetSize(m.width-10-2, m.height/2)
		case tea.KeyMsg:
			if m.list.FilterState() == list.Filtering{
				nl, c := m.list.Update(msg)
				m.list = nl
				return m, c
			}
			switch msg.Type{
			case tea.KeyEnter:
				if m.listMultiSelect == true{
					for _, item := range m.list.Items(){
						v, ok := item.(popUpListItem)
						if !ok{
							continue
						}
						if v.pu_selected{
							m.values = append(m.values, v.Url)
						}
					}
				} else {
					selected := m.list.SelectedItem()

					sel, ok := selected.(popUpListItem)
					if ok{
						m.values = append(m.values, sel.Title_Field)
						m.values = append(m.values, sel.Url)
					}

				}
				m.action(m.config,m.values)
				return m.prevModel, RefreshCmd("")

			case tea.KeySpace:
				idx := m.list.Index()
				if idx >= 0 {
					if it, ok := m.list.Items()[idx].(popUpListItem); ok {
						it.pu_selected = !it.pu_selected        
						m.list.SetItem(idx, it)           
					}
				}
			case tea.KeyEsc:
				return m.prevModel, tea.WindowSize()

			default:
				if in(msg.String(), m.config.Keys["quit"]){
					return m.prevModel, tea.WindowSize()
				}
				nl, c := m.list.Update(msg)
				m.list = nl
				return m, c
			}

		default:
			nl, c := m.list.Update(msg)
			m.list = nl
			return m, c

		}
	}
	return m, nil
}

type popUpListItem struct{
	Title_Field  string `json:"Title"`
	Url  string `json:"Url"`
	pu_selected bool
}

func (i popUpListItem) Title () string { return i.Title_Field}
func (i popUpListItem) Description () string {return  ""}
func (i popUpListItem) FilterValue () string {return i.Title_Field}

type FeediePopUpDelegate struct{
	list.DefaultDelegate
	config FeedieConfig
	multi bool
}
func (d FeediePopUpDelegate) Height() int {return 1}
func (d FeediePopUpDelegate) Spacing() int {return 1}
func (d FeediePopUpDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(popUpListItem)
	if ok {
		bar := " " 
		sel_mark :="[ ]"
		title := i.Title()

		if i.pu_selected{
			sel_mark = "[*]"
		}


		if index == m.Index() {
			bar = "┃" // custom indicator
			title = d.Styles.SelectedTitle.MaxWidth(m.Width()-2).Render(title)
			bar = d.Styles.SelectedTitle.Foreground(lipgloss.Color(d.config.FocusBorderC)).Render(bar)
		}else{
			bar = " " // custom indicator
			title = d.Styles.NormalTitle.MaxWidth(m.Width() - 2).Render(title)
			bar = d.Styles.NormalTitle.Foreground(lipgloss.Color(d.config.SelectCursor)).Render(bar)
		}
		if d.multi{
			fmt.Fprintf(w, "%s%s%s", bar,sel_mark, title)
		}else{
			fmt.Fprintf(w, "%s%s", bar, title)

		}
	}

}
