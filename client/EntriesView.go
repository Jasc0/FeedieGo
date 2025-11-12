package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const PRELOAD_AMT = 5


type entriesModel struct {
	prevModel tea.Model
	config FeedieConfig
	src func(FeedieConfig)[]list_entry
	width, height int
	ready bool
	focusL bool;
	vp viewport.Model
	list list.Model
	thumbnail thumbnailManager
}

func (m entriesModel) getSelectedEntry() list_entry {
	if target := m.list.SelectedItem(); target != nil{
		if target, ok := target.(list_entry); ok{
			return target
		}
	}
	return list_entry{}
}
func getEntryKeys(fc FeedieConfig) func() []key.Binding{
	entryCommands := []string{"changeFocus", "feedMenu", "openMenu", "open"}	
	ret := []key.Binding{}

	for command, keys := range fc.Keys{
		if in(command, entryCommands){
			kb := key.NewBinding(
				key.WithKeys(keys...),
				key.WithHelp(strings.Join(keys,"\\"),command),
			)
			ret = append(ret, kb)
		}


	}
	return func () []key.Binding {return ret}
}

func initialEntriesModel( f func(FeedieConfig)[]list_entry, c FeedieConfig, prev tea.Model, initial []list_entry) entriesModel {
	
	m := entriesModel{
		prevModel: prev,
		config: c,
		src : f,
		width: DEFAULT_W,
		height: DEFAULT_H,
		ready: false,
		focusL: true,
		vp: viewport.New(DEFAULT_H, DEFAULT_W),
		list: list.New([]list.Item{}, c.getEntryDelegate(),
		DEFAULT_W , DEFAULT_H),
		thumbnail: initThumbnailManager(c),
	}

	m.vp.MouseWheelEnabled = true
	m.list.SetShowTitle(false)
	m.list.SetShowHelp(false)
	m.list.Help.Ellipsis=""
	m.list.Help.ShowAll = true
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
	m.list.AdditionalFullHelpKeys = getEntryKeys(c)


	var entries []list.Item
	if len(initial) > 0{
		for _, en := range initial{
			entries = append(entries, en)
		}
	} else{
		for _, en := range m.src(m.config){
			entries = append(entries, en)
		}
	}
	m.list.SetItems(entries)
	go m.preloadThumbnails(PRELOAD_AMT)

	return m
}

func (m entriesModel) preloadThumbnails(preload int){
	items := m.list.Items()
	si := max(0, m.list.Index() - preload/2)
	ei := min(si+preload, si+preload, len(items))
	urls := []string{}
	for _, it := range items[si:ei]{
		it := it.(list_entry)
		urls = append(urls, it.Thumbnail)
	}
	m.thumbnail.preloadImages(urls)

}

func (m entriesModel) Init() tea.Cmd { 
	m.ready = false
	return nil
}
func (m entriesModel) Refresh() tea.Cmd {
	var entries []list.Item
	for _, en := range m.src(m.config){
		entries = append(entries, en)
	}
	return m.list.SetItems(entries)
}

func getPaneWidth(window_width int) int{
	ratio := 1.0/2
	return int(float64(window_width)*ratio-2)
}
func getPaneHeight(window_height int, ratio float64) int{
	return int(float64(window_height)*ratio-(2))
}

func (m entriesModel) View() string {
	if !m.ready{
		return "loading... \n"
	}
	cur := m.getSelectedEntry()


	innerW := max(20, m.width)
	innerH := max(10, m.height-2)
	leftPaneW := getPaneWidth(innerW)
	rightPaneW := getPaneWidth(innerW)

	var upper_right_height, lower_right_height int
	var lStyle,rStyle lipgloss.Style
	if m.focusL{
		lStyle = m.config.getFocusedStyle()
		rStyle = m.config.getNormalStyle()
	} else{
		lStyle = m.config.getNormalStyle()
		rStyle = m.config.getFocusedStyle()
	}
	if cur.Thumbnail != ""{
		lower_right_height = getPaneHeight(m.height, 1 - m.config.ThumbnailRatio)
		upper_right_height = getPaneHeight(m.height, m.config.ThumbnailRatio)
		if (lower_right_height + upper_right_height < innerH){
			lower_right_height += innerH - lower_right_height - upper_right_height -2

		}
	} else{
		lower_right_height = getPaneHeight(m.height, 1)
		upper_right_height = getPaneHeight(m.height, 0)
	}

	left := lStyle.Width(leftPaneW).
		Height(getPaneHeight(m.height,1)).
		Render(m.list.View())
	upper_right := rStyle.Width(rightPaneW).
		Height(upper_right_height).
		Render("")
	lower_right := rStyle.Width(rightPaneW).
		Height(lower_right_height).
		Render(m.vp.View())
	right := lipgloss.JoinVertical(lipgloss.Bottom,upper_right,lower_right)
	return lipgloss.JoinHorizontal(lipgloss.Bottom,left, right)
	
}

func (m entriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cur := m.getSelectedEntry()
	switch msg := msg.(type) {
	case FeedieMsg:
		return m, m.Refresh()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(getPaneWidth(m.width),getPaneHeight(m.height, 1))
		m.vp.Width = getPaneWidth(m.width)
		m.vp.Height = getPaneHeight(m.height, 1 - m.config.ThumbnailRatio )
		m.ready = true
		cur := m.getSelectedEntry()
		m.vp.SetContent(cur.FullDescription(m.vp.Width))


	case tea.KeyMsg:
		k := msg.String()
		if in(k,m.config.Keys["quit"]){
			if m.list.FilterState() != list.Filtering{
				return m, tea.Quit
			}
		}
		if in(k,m.config.Keys["changeFocus"]){
			m.focusL = !m.focusL
			return m, nil
		}
			
		if in(k,m.config.Keys["cursorDown"]){
			if m.list.FilterState() != list.Filtering{
				if m.focusL{
					m.vp.YOffset=0
					m.vp.SetXOffset(0)
					m.list.CursorDown()
					cur = m.getSelectedEntry()
					m.vp.SetContent(cur.FullDescription(m.vp.Width))
					m.drawCurImage()
					if cur.Thumbnail != ""{
						m.vp.Height = getPaneHeight(m.height,1 - m.config.ThumbnailRatio)
					} else{
						m.vp.Height = getPaneHeight(m.height, 1)
					}
					return m, nil

				} else{
					m.vp.ScrollDown(1)

				}
				

			}
		}
		if in(k,m.config.Keys["cursorUp"]){
			if m.list.FilterState() != list.Filtering{
				if m.focusL{
					m.vp.YOffset=0
					m.vp.SetXOffset(0)
					m.list.CursorUp()
					cur = m.getSelectedEntry()
					m.vp.SetContent(cur.FullDescription(m.vp.Width))
					m.drawCurImage()
					if cur.Thumbnail != ""{
						m.vp.Height = getPaneHeight(m.height,1-m.config.ThumbnailRatio)
					} else{
						m.vp.Height = getPaneHeight(m.height, 1)
					}
					return m, nil

				} else{
					m.vp.ScrollUp(1)

				}
			}
		}
		if in(k,m.config.Keys["filter"]){
			nl, c := m.list.Update(msg)
			m.list = nl
			return m,c
		}
		if in(k,m.config.Keys["feedMenu"]){
			if m.list.FilterState() != list.Filtering{
				m.thumbnail.clear()
				return m.prevModel, RefreshCmd("")
			}
		}
		if in(k,m.config.Keys["openMenu"]){
			if m.list.FilterState() != list.Filtering{
				m.thumbnail.clear()
				return initialListPopupModel(m.config, m.config.getLinkOpener, cur.getLinks,
					false, m, "Choose which link to open", []string{}, RefreshCmd), tea.WindowSize()

			}
		}
		if in(k,m.config.Keys["open"]){
			if m.list.FilterState() != list.Filtering{
				if len(cur.Links) >= 1{
					defaultLink := cur.Links[0]
					m.config.getLinkOpener(m.config, []string{defaultLink.URL, defaultLink.Type})
				}
			}
		}
		if in(k,m.config.Keys["refresh"]){
			if m.list.FilterState() != list.Filtering{
				return m, RefreshCmd("")
			}
		}
		if in(k,m.config.Keys["help"]){
			if m.list.ShowHelp(){
				m.list.SetShowHelp(false)
			} else{
				m.list.SetShowHelp(true)
			}
			return m,nil
		}
		if m.focusL{
			nl, c := m.list.Update(msg)
			m.list = nl
			return m,c
		} else{
			nvp, c := m.vp.Update(msg)
			m.vp = nvp
			return m,c
		}
	case tea.MouseMsg:
		if m.focusL{

			nl, c := m.list.Update(msg)
			m.list = nl
			return m,c
		} else{
			nvp, c := m.vp.Update(msg)
			m.vp = nvp
			return m,c
		}
	default:
		if m.focusL{
			nl, c := m.list.Update(msg)
			m.list = nl
			return m,c
		} else{
			nvp, c := m.vp.Update(msg)
			m.vp = nvp
			return m,c
		}

	}
	m.drawCurImage()
	if cur.Thumbnail != ""{
		m.vp.Height = getPaneHeight(m.height,1-m.config.ThumbnailRatio)
	} else{
		m.vp.Height = getPaneHeight(m.height, 1)
	}
	return m,nil
}


func (m entriesModel) drawCurImage(){
	cur := m.getSelectedEntry()
	if cur.Thumbnail != ""{
		x := m.list.Width()+3
		y := 1
		width := m.vp.Width
		height := getPaneHeight(m.height,m.config.ThumbnailRatio)
		if ok := m.thumbnail.drawImage(x,y,width,height, cur.Thumbnail); !ok{
			m.vp.Height = getPaneHeight(m.height,1)
		}
	}else{
		m.thumbnail.clear()
	} 
	go m.preloadThumbnails(PRELOAD_AMT)

}
