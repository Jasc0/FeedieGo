package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const preloadAmt = 5

type entriesModel struct {
	prevModel     tea.Model
	config        FeedieConfig
	src           func(FeedieConfig, int) []list_entry
	width, height int
	ready         bool
	listFocused   bool
	vp            viewport.Model
	list          list.Model
	thumbnail     thumbnailManager
	maxPageOffset int
}

func (m entriesModel) getSelectedEntry() list_entry {
	if target := m.list.SelectedItem(); target != nil {
		if entry, ok := target.(list_entry); ok {
			return entry
		}
	}
	return list_entry{}
}

func getEntryKeys(config FeedieConfig) func() []key.Binding {
	entryCommands := []string{"changeFocus", "feedMenu", "openMenu", "open"}
	ret := []key.Binding{}
	for command, keys := range config.Keys {
		if in(command, entryCommands) {
			binding := key.NewBinding(
				key.WithKeys(keys...),
				key.WithHelp(strings.Join(keys, "\\"), command),
			)
			ret = append(ret, binding)
		}
	}
	return func() []key.Binding { return ret }
}

func initialEntriesModel(src func(FeedieConfig, int) []list_entry, config FeedieConfig, prev tea.Model, initial []list_entry) entriesModel {
	m := entriesModel{
		prevModel:   prev,
		config:      config,
		src:         src,
		width:       defaultW,
		height:      defaultH,
		ready:       false,
		listFocused: true,
		vp:          viewport.New(defaultH, defaultW),
		list:        list.New([]list.Item{}, config.getEntryDelegate(), defaultW, defaultH),
		thumbnail:   initThumbnailManager(config),
	}

	m.vp.MouseWheelEnabled = true
	m.list.SetShowTitle(false)
	m.list.SetShowHelp(false)
	m.list.Help.Ellipsis = ""
	m.list.Help.ShowAll = true
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
	m.list.AdditionalFullHelpKeys = getEntryKeys(config)

	var entries []list.Item
	if len(initial) > 0 {
		for _, entry := range initial {
			entries = append(entries, entry)
		}
	} else {
		for _, entry := range m.src(m.config, 0) {
			entries = append(entries, entry)
		}
	}
	m.list.SetItems(entries)
	go m.preloadThumbnails(preloadAmt)

	return m
}

func (m entriesModel) preloadThumbnails(preload int) {
	items := m.list.Items()
	startIdx := max(0, m.list.Index()-preload/2)
	endIdx := min(startIdx+preload, len(items))
	urls := []string{}
	for _, item := range items[startIdx:endIdx] {
		entry := item.(list_entry)
		urls = append(urls, entry.Thumbnail)
	}
	m.thumbnail.preloadImages(urls)
}

func (m entriesModel) Init() tea.Cmd {
	m.ready = false
	return nil
}

func (m entriesModel) Refresh() (entriesModel, tea.Cmd) {
	var entries []list.Item
	for page := 0; page <= m.maxPageOffset; page++ {
		for _, entry := range m.src(m.config, page) {
			entries = append(entries, entry)
		}
	}
	cmd := m.list.SetItems(entries)
	return m, cmd
}

func getPaneWidth(windowWidth int) int {
	return int(float64(windowWidth)*0.5 - 2)
}

func getPaneHeight(windowHeight int, ratio float64) int {
	return int(float64(windowHeight)*ratio - 2)
}

func (m entriesModel) View() string {
	if !m.ready {
		return "loading... \n"
	}
	selected := m.getSelectedEntry()

	innerW := max(20, m.width)
	innerH := max(10, m.height-2)
	leftPaneW := getPaneWidth(innerW)
	rightPaneW := getPaneWidth(innerW)

	var upperRightH, lowerRightH int
	var leftStyle, rightStyle lipgloss.Style
	if m.listFocused {
		leftStyle = m.config.getFocusedStyle()
		rightStyle = m.config.getNormalStyle()
	} else {
		leftStyle = m.config.getNormalStyle()
		rightStyle = m.config.getFocusedStyle()
	}
	if selected.Thumbnail != "" {
		lowerRightH = getPaneHeight(m.height, 1-m.config.ThumbnailRatio)
		upperRightH = getPaneHeight(m.height, m.config.ThumbnailRatio)
		if lowerRightH+upperRightH < innerH {
			lowerRightH += innerH - lowerRightH - upperRightH - 2
		}
	} else {
		lowerRightH = getPaneHeight(m.height, 1)
		upperRightH = getPaneHeight(m.height, 0)
	}

	left := leftStyle.Width(leftPaneW).
		Height(getPaneHeight(m.height, 1)).
		Render(m.list.View())
	upperRight := rightStyle.Width(rightPaneW).
		Height(upperRightH).
		Render("")
	lowerRight := rightStyle.Width(rightPaneW).
		Height(lowerRightH).
		Render(m.vp.View())
	right := lipgloss.JoinVertical(lipgloss.Bottom, upperRight, lowerRight)
	return lipgloss.JoinHorizontal(lipgloss.Bottom, left, right)
}

func (m entriesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	selected := m.getSelectedEntry()
	switch msg := msg.(type) {
	case FeedieMsg:
		m, cmd := m.Refresh()
		return m, cmd
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(getPaneWidth(m.width), getPaneHeight(m.height, 1))
		m.vp.Width = getPaneWidth(m.width)
		m.vp.Height = getPaneHeight(m.height, 1-m.config.ThumbnailRatio)
		m.ready = true
		selected := m.getSelectedEntry()
		m.vp.SetContent(selected.FullDescription(m.vp.Width))

	case tea.KeyMsg:
		k := msg.String()
		if in(k, m.config.Keys["quit"]) {
			if m.list.FilterState() != list.Filtering {
				return m, tea.Quit
			}
		}
		if in(k, m.config.Keys["changeFocus"]) {
			m.listFocused = !m.listFocused
			return m, nil
		}
		if in(k, m.config.Keys["cursorDown"]) {
			if m.list.FilterState() != list.Filtering {
				if m.listFocused {
					m.vp.YOffset = 0
					m.vp.SetXOffset(0)
					m.list.CursorDown()
					selected = m.getSelectedEntry()
					m.vp.SetContent(selected.FullDescription(m.vp.Width))
					m.drawCurImage()
					if selected.Thumbnail != "" {
						m.vp.Height = getPaneHeight(m.height, 1-m.config.ThumbnailRatio)
					} else {
						m.vp.Height = getPaneHeight(m.height, 1)
					}
					return m, m.paginationLogic()
				} else {
					m.vp.ScrollDown(1)
				}
			}
		}
		if in(k, m.config.Keys["cursorUp"]) {
			if m.list.FilterState() != list.Filtering {
				if m.listFocused {
					m.vp.YOffset = 0
					m.vp.SetXOffset(0)
					m.list.CursorUp()
					selected = m.getSelectedEntry()
					m.vp.SetContent(selected.FullDescription(m.vp.Width))
					m.drawCurImage()
					if selected.Thumbnail != "" {
						m.vp.Height = getPaneHeight(m.height, 1-m.config.ThumbnailRatio)
					} else {
						m.vp.Height = getPaneHeight(m.height, 1)
					}
					return m, nil
				} else {
					m.vp.ScrollUp(1)
				}
			}
		}
		if in(k, m.config.Keys["filter"]) {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		}
		if in(k, m.config.Keys["feedMenu"]) {
			if m.list.FilterState() != list.Filtering {
				m.thumbnail.clear()
				return m.prevModel, RefreshCmd("")
			}
		}
		if in(k, m.config.Keys["openMenu"]) {
			if m.list.FilterState() != list.Filtering {
				m.thumbnail.clear()
				return initialListPopupModel(m.config, m.config.getLinkOpener, selected.getLinks,
					false, m, "Choose which link to open", []string{}, RefreshCmd), tea.WindowSize()
			}
		}
		if in(k, m.config.Keys["open"]) {
			if m.list.FilterState() != list.Filtering {
				if len(selected.Links) >= 1 {
					defaultLink := selected.Links[0]
					m.config.getLinkOpener(m.config, []string{defaultLink.URL, defaultLink.Type})
				}
			}
		}
		if in(k, m.config.Keys["copyLink"]) {
			if m.list.FilterState() != list.Filtering {
				if len(selected.Links) >= 1 {
					defaultLink := selected.Links[0]
					m.config.getYanker(m.config, []string{defaultLink.URL})
				}
			}
		}
		if in(k, m.config.Keys["refresh"]) {
			if m.list.FilterState() != list.Filtering {
				return m, RefreshCmd("")
			}
		}
		if in(k, m.config.Keys["help"]) {
			if m.list.ShowHelp() {
				m.list.SetShowHelp(false)
			} else {
				m.list.SetShowHelp(true)
			}
			return m, nil
		}
		if m.listFocused {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, tea.Batch(cmd, m.paginationLogic())
		} else {
			newVP, cmd := m.vp.Update(msg)
			m.vp = newVP
			return m, cmd
		}
	case tea.MouseMsg:
		if m.listFocused {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		} else {
			newVP, cmd := m.vp.Update(msg)
			m.vp = newVP
			return m, cmd
		}
	default:
		if m.listFocused {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		} else {
			newVP, cmd := m.vp.Update(msg)
			m.vp = newVP
			return m, cmd
		}
	}
	m.drawCurImage()
	if selected.Thumbnail != "" {
		m.vp.Height = getPaneHeight(m.height, 1-m.config.ThumbnailRatio)
	} else {
		m.vp.Height = getPaneHeight(m.height, 1)
	}
	return m, nil
}

func (m entriesModel) drawCurImage() {
	selected := m.getSelectedEntry()
	if selected.Thumbnail != "" {
		x := m.list.Width() + 3
		y := 1
		width := m.vp.Width
		height := getPaneHeight(m.height, m.config.ThumbnailRatio)
		if ok := m.thumbnail.drawImage(x, y, width, height, selected.Thumbnail); !ok {
			m.vp.Height = getPaneHeight(m.height, 1)
		}
	} else {
		m.thumbnail.clear()
	}
	go m.preloadThumbnails(preloadAmt)
}

func (m *entriesModel) paginationLogic() tea.Cmd {
	if m.list.Paginator.OnLastPage() {
		current := m.list.Items()
		nextPage := m.src(m.config, m.maxPageOffset+1)
		if len(nextPage) == 0 {
			return nil
		}
		m.maxPageOffset++
		for _, entry := range nextPage {
			current = append(current, entry)
		}
		return m.list.SetItems(current)
	}
	return nil
}
