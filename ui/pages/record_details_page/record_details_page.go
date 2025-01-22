package record_details_page

import (
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ktea/kadmin"
	"ktea/kontext"
	"ktea/styles"
	"ktea/ui"
	"ktea/ui/components/statusbar"
	ktable "ktea/ui/components/table"
	"ktea/ui/pages/nav"
	"sort"
	"strconv"
	"strings"
	"time"
)

type focus bool

const (
	payloadFocus focus = true
	headersFocus focus = false
)

type Model struct {
	record         *kadmin.ConsumerRecord
	payloadVp      *viewport.Model
	headerValueVp  *viewport.Model
	topic          *kadmin.Topic
	headerKeyTable *table.Model
	headerRows     []table.Row
	focus          focus
	payload        string
	metaInfo       string
}

func (m *Model) View(ktx *kontext.ProgramKtx, renderer *ui.Renderer) string {
	contentStyle, headersTableStyle := m.determineStyles()

	payloadWidth := int(float64(ktx.WindowWidth) * 0.70)
	height := ktx.AvailableHeight - 2

	m.createPayloadViewPort(payloadWidth, height)

	headerSideBar := m.createSidebar(ktx, payloadWidth, height, headersTableStyle)

	return lipgloss.NewStyle().
		Render(lipgloss.JoinHorizontal(
			lipgloss.Top,
			renderer.RenderWithStyle(m.payloadVp.View(), contentStyle),
			headerSideBar,
		))
}

func (m *Model) createSidebar(ktx *kontext.ProgramKtx, payloadWidth int, height int, headersTableStyle lipgloss.Style) string {
	sideBarWidth := ktx.WindowWidth - (payloadWidth + 7)

	var headerSideBar string
	if len(m.record.Headers) == 0 {
		headerSideBar = ui.JoinVertical(
			lipgloss.Top,
			lipgloss.NewStyle().Padding(1).Render(m.metaInfo),
			lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Padding(1).Render("No headers present")),
		)
	} else {
		headerValueTableHeight := len(m.record.Headers) + 4

		headerValueVp := viewport.New(sideBarWidth, height-headerValueTableHeight-4)
		m.headerValueVp = &headerValueVp
		m.headerKeyTable.SetColumns([]table.Column{
			{"Header Key", sideBarWidth},
		})
		m.headerKeyTable.SetHeight(headerValueTableHeight)
		m.headerKeyTable.SetRows(m.headerRows)

		headerValueLine := strings.Builder{}
		for i := 0; i < sideBarWidth; i++ {
			headerValueLine.WriteString("─")
		}

		var headerValue string
		selectedRow := m.headerKeyTable.SelectedRow()
		if selectedRow == nil {
			if len(m.record.Headers) > 0 {
				headerValue = m.record.Headers[0].Value
			}
		} else {
			headerValue = m.record.Headers[m.headerKeyTable.Cursor()].Value
		}
		m.headerValueVp.SetContent("Header Value\n" + headerValueLine.String() + "\n" + headerValue)

		headerSideBar = ui.JoinVertical(
			lipgloss.Top,
			lipgloss.NewStyle().Padding(1).Render(m.metaInfo),
			headersTableStyle.Render(lipgloss.JoinVertical(lipgloss.Top, m.headerKeyTable.View(), m.headerValueVp.View())),
		)
	}
	return headerSideBar
}

func (m *Model) createPayloadViewPort(payloadWidth int, height int) {
	if m.payloadVp == nil {
		payloadVp := viewport.New(payloadWidth, height)
		m.payloadVp = &payloadVp
	} else {
		m.payloadVp.Height = height
		m.payloadVp.Width = payloadWidth
	}
	m.payloadVp.SetContent(m.payload)
}

func (m *Model) determineStyles() (lipgloss.Style, lipgloss.Style) {
	var contentStyle lipgloss.Style
	var headersTableStyle lipgloss.Style
	if m.focus == payloadFocus {
		contentStyle = lipgloss.NewStyle().
			Inherit(styles.TextViewPort).
			BorderForeground(lipgloss.Color(styles.ColorFocusBorder))
		headersTableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(0).
			Margin(0).
			BorderForeground(lipgloss.Color(styles.ColorBlurBorder))
	} else {
		contentStyle = lipgloss.NewStyle().
			Inherit(styles.TextViewPort).
			BorderForeground(lipgloss.Color(styles.ColorBlurBorder))
		headersTableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(0).
			Margin(0).
			BorderForeground(lipgloss.Color(styles.ColorFocusBorder))
	}
	return contentStyle, headersTableStyle
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if m.payloadVp == nil {
		return nil
	}

	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return ui.PublishMsg(nav.LoadCachedConsumptionPageMsg{})
		case "ctrl+h", "left", "right":
			m.focus = !m.focus
		case "c":
			if m.focus == payloadFocus {
				err := clipboard.WriteAll(m.record.Value)
				if err != nil {
					return nil
				}
			} else {
			}
		default:
			cmds = m.updatedFocussedArea(msg, cmds)
		}
	}

	return tea.Batch(cmds...)
}

func (m *Model) updatedFocussedArea(msg tea.Msg, cmds []tea.Cmd) []tea.Cmd {
	if m.focus == payloadFocus {
		vp, cmd := m.payloadVp.Update(msg)
		cmds = append(cmds, cmd)
		m.payloadVp = &vp
	} else {
		t, cmd := m.headerKeyTable.Update(msg)
		cmds = append(cmds, cmd)
		m.headerKeyTable = &t
	}
	return cmds
}

func (m *Model) Shortcuts() []statusbar.Shortcut {
	whatToCopy := "Header Value"
	if m.focus == payloadFocus {
		whatToCopy = "Content"
	}
	return []statusbar.Shortcut{
		{"Toggle Headers/Content", "C-h/Arrows"},
		{"Go Back", "esc"},
		{"Copy " + whatToCopy, "c"},
	}
}

func (m *Model) Title() string {
	return "Topics / " + m.topic.Name + " / Records / " + strconv.FormatInt(m.record.Offset, 10)
}

func New(record *kadmin.ConsumerRecord, topic *kadmin.Topic) *Model {
	headersTable := ktable.NewDefaultTable()

	var headerRows []table.Row
	sort.SliceStable(record.Headers, func(i, j int) bool {
		return record.Headers[i].Key < record.Headers[j].Key
	})
	for _, header := range record.Headers {
		headerRows = append(headerRows, table.Row{header.Key})
	}

	payload := ui.PrettyPrintJson(record.Value)

	key := record.Key
	if key == "" {
		key = "<null>"
	}

	metaInfo := fmt.Sprintf("key: %s\ntimestamp: %s", key, record.Timestamp.Format(time.UnixDate))

	return &Model{
		record:         record,
		topic:          topic,
		headerKeyTable: &headersTable,
		focus:          payloadFocus,
		headerRows:     headerRows,
		payload:        payload,
		metaInfo:       metaInfo,
	}
}
