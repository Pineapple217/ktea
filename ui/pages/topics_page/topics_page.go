package topics_page

import (
	"context"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ktea/kadmin"
	"ktea/kontext"
	"ktea/styles"
	"ktea/ui"
	"ktea/ui/components/cmdbar"
	"ktea/ui/components/notifier"
	"ktea/ui/components/statusbar"
	"ktea/ui/pages/nav"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type state int

type Model struct {
	topics        []kadmin.Topic
	table         table.Model
	shortcuts     []statusbar.Shortcut
	cmdBar        *cmdbar.TableCmdsBar[string]
	rows          []table.Row
	moveCursor    func()
	lister        kadmin.TopicLister
	ctx           context.Context
	tableFocussed bool
}

func (m *Model) View(ktx *kontext.ProgramKtx, renderer *ui.Renderer) string {
	var views []string
	cmdBarView := m.cmdBar.View(ktx, renderer)
	views = append(views, cmdBarView)

	m.table.SetHeight(ktx.AvailableHeight - 2)
	m.table.SetWidth(ktx.WindowWidth - 2)
	m.table.SetColumns([]table.Column{
		{"Name", int(float64(ktx.WindowWidth-9) * 0.7)},
		{"Partitions", int(float64(ktx.WindowWidth-9) * 0.1)},
		{"Replicas", int(float64(ktx.WindowWidth-9) * 0.1)},
		{"In Sync Replicas", int(float64(ktx.WindowWidth-9) * 0.1)},
	})
	m.table.SetRows(m.rows)

	if m.moveCursor != nil {
		m.moveCursor()
	}

	var tableView string
	if m.tableFocussed {
		tableView = renderer.RenderWithStyle(m.table.View(), styles.Table.Focus)
	} else {
		tableView = renderer.RenderWithStyle(m.table.View(), styles.Table.Blur)
	}
	views = append(views, tableView)

	return ui.JoinVertical(lipgloss.Top, views...)
}
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 2)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.topics == nil {
			return nil
		}
		switch msg.String() {
		case "ctrl+n":
			return ui.PublishMsg(nav.LoadCreateTopicPageMsg{})
		case "ctrl+o":
			return ui.PublishMsg(nav.LoadTopicConfigPageMsg{})
		case "ctrl+p":
			return ui.PublishMsg(nav.LoadPublishPageMsg{Topic: m.SelectedTopic()})
		case "f5":
			m.topics = nil
			return m.lister.ListTopics
		case "enter":
			// only accept enter when the table is focussed
			if !m.cmdBar.IsFocussed() {
				// TODO ignore enter when there are no topics loaded
				return ui.PublishMsg(nav.LoadConsumptionFormPageMsg{
					Topic: m.SelectedTopic(),
				})
			}
		}
	case spinner.TickMsg:
		selectedTopic := m.SelectedTopicName()
		_, c := m.cmdBar.Update(msg, &selectedTopic)
		if c != nil {
			cmds = append(cmds, c)
		}
	case kadmin.TopicDeletionStartedMsg:
		cmds = append(cmds, msg.AwaitCompletion)
	case kadmin.TopicListingStartedMsg:
		cmds = append(cmds, msg.AwaitCompletion)
	case kadmin.TopicListedMsg:
		m.topics = msg.Topics
	case kadmin.TopicDeletedMsg:
		m.topics = slices.DeleteFunc(
			m.topics,
			func(t kadmin.Topic) bool { return msg.TopicName == t.Name },
		)
	}

	name := m.SelectedTopicName()
	msg, cmd := m.cmdBar.Update(msg, &name)
	m.tableFocussed = !m.cmdBar.IsFocussed()
	cmds = append(cmds, cmd)

	m.rows = m.filterTopicsBySearchTerm()

	// make sure table navigation is off when the cmdbar is focussed
	if !m.cmdBar.IsFocussed() {
		t, cmd := m.table.Update(msg)
		m.table = t
		cmds = append(cmds, cmd)
	}

	if m.cmdBar.HasSearchedAtLeastOneChar() {
		m.table.GotoTop()
	}
	return tea.Batch(cmds...)
}

func (m *Model) filterTopicsBySearchTerm() []table.Row {
	var rows []table.Row
	for _, topic := range m.topics {
		if m.cmdBar.GetSearchTerm() != "" {
			if strings.Contains(strings.ToLower(topic.Name), strings.ToLower(m.cmdBar.GetSearchTerm())) {
				rows = append(
					rows,
					table.Row{
						topic.Name,
						strconv.Itoa(topic.Partitions),
						strconv.Itoa(topic.Replicas),
						"N/A",
					},
				)
			}
		} else {
			rows = append(
				rows,
				table.Row{
					topic.Name,
					strconv.Itoa(topic.Partitions),
					strconv.Itoa(topic.Replicas),
					"N/A",
				},
			)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})
	return rows
}

func (m *Model) SelectedTopic() *kadmin.Topic {
	selectedTopic := m.SelectedTopicName()
	for _, t := range m.topics {
		if t.Name == selectedTopic {
			return &t
		}
	}
	panic("selected topic not found")
}

func (m *Model) SelectedTopicName() string {
	selectedRow := m.table.SelectedRow()
	var selectedTopic string
	if selectedRow != nil {
		selectedTopic = selectedRow[0]
	}
	return selectedTopic
}

func (m *Model) Title() string {
	return "Topics"
}

func (m *Model) Shortcuts() []statusbar.Shortcut {
	return m.shortcuts
}

func (m *Model) Refresh() tea.Cmd {
	m.topics = nil
	return m.lister.ListTopics
}

func New(topicDeleter kadmin.TopicDeleter, lister kadmin.TopicLister) (*Model, tea.Cmd) {
	var m = Model{}
	m.shortcuts = []statusbar.Shortcut{
		{"Search", "/"},
		{"Consume", "enter"},
		{"Publish", "C-p"},
		{"Create", "C-n"},
		{"Delete", "F2"},
		{"Configs", "C-o"},
		{"Refresh", "F5"},
	}
	m.table = table.New(
		table.WithFocused(true),
		table.WithStyles(styles.Table.Styles),
	)
	m.table.SetColumns([]table.Column{
		{"Name", 1},
		{"Partitions", 1},
		{"Replicas", 1},
		{"In Sync Replicas", 1},
	})

	deleteMsgFunc := func(topic string) string {
		message := topic + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7571F9")).
			Bold(true).
			Render(" will be deleted permanently")
		return message
	}

	deleteFunc := func(topic string) tea.Cmd {
		return func() tea.Msg {
			return topicDeleter.DeleteTopic(topic)
		}
	}

	notifierCmdBar := cmdbar.NewNotifierCmdBar()

	cmdbar.WithMsgHandler(
		notifierCmdBar,
		func(
			msg kadmin.TopicListingStartedMsg,
			m *notifier.Model,
		) (bool, tea.Cmd) {
			cmd := m.SpinWithLoadingMsg("Loading Topics")
			return true, cmd
		},
	)

	cmdbar.WithMsgHandler(
		notifierCmdBar,
		func(
			msg kadmin.TopicListedMsg,
			m *notifier.Model,
		) (bool, tea.Cmd) {
			m.Idle()
			return true, m.AutoHideCmd()
		},
	)

	cmdbar.WithMsgHandler(
		notifierCmdBar,
		func(
			msg kadmin.TopicListedErrorMsg,
			m *notifier.Model,
		) (bool, tea.Cmd) {
			m.ShowErrorMsg("Error listing Topics", msg.Err)
			return true, nil
		},
	)

	cmdbar.WithMsgHandler(
		notifierCmdBar,
		func(
			msg kadmin.TopicDeletedMsg,
			m *notifier.Model,
		) (bool, tea.Cmd) {
			m.ShowSuccessMsg("Topic Deleted")
			return true, m.AutoHideCmd()
		},
	)

	cmdbar.WithMsgHandler(
		notifierCmdBar,
		func(
			msg kadmin.TopicDeletionStartedMsg,
			m *notifier.Model,
		) (bool, tea.Cmd) {
			cmd := m.SpinWithLoadingMsg("Deleting Topic")
			return true, cmd
		},
	)

	cmdbar.WithMsgHandler(
		notifierCmdBar,
		func(
			msg kadmin.TopicDeletionErrorMsg,
			m *notifier.Model,
		) (bool, tea.Cmd) {
			m.ShowErrorMsg("Error Deleting Topic", msg.Err)
			return true, m.AutoHideCmd()
		},
	)

	m.cmdBar = cmdbar.NewTableCmdsBar[string](
		cmdbar.NewDeleteCmdBar(deleteMsgFunc, deleteFunc, nil),
		cmdbar.NewSearchCmdBar("Search topics by name"),
		notifierCmdBar,
	)
	m.lister = lister
	return &m, lister.ListTopics
}
