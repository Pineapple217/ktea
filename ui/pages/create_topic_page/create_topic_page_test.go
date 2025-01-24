package create_topic_page

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"ktea/kadmin"
	"ktea/kontext"
	"ktea/tests/keys"
	"ktea/ui"
	"ktea/ui/pages/nav"
	"testing"
)

func batchUpdate(m *Model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	cmd = m.Update(msg)
	msg = cmd()
	cmd = m.Update(msg)
}

func CreateTopicSectionWithCursorAtPartitionsField() *Model {
	m := New(&MockTopicCreator{})
	cmd := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a'},
		Alt:   false,
		Paste: false,
	})
	batchUpdate(m, cmd)
	cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	batchUpdate(m, cmd)
	return m
}

type MockTopicCreator struct {
	CreateTopicFunc func(details kadmin.TopicCreationDetails) tea.Msg
}

func (m *MockTopicCreator) CreateTopic(tcd kadmin.TopicCreationDetails) tea.Msg {
	if m.CreateTopicFunc != nil {
		return m.CreateTopicFunc(tcd)
	}
	return nil
}

func TestCreateTopic(t *testing.T) {

	type CapturedTopicCreationDetails struct {
		kadmin.TopicCreationDetails
	}

	t.Run("esc", func(t *testing.T) {
		mockCreator := MockTopicCreator{
			CreateTopicFunc: func(details kadmin.TopicCreationDetails) tea.Msg {
				if details.Name == "" {
					return errors.New("topic name cannot be empty")
				}
				return CapturedTopicCreationDetails{details}
			},
		}
		m := New(&mockCreator)

		t.Run("goes back to topic list page", func(t *testing.T) {
			cmd := m.Update(keys.Key(tea.KeyEsc))

			assert.Equal(t, nav.LoadTopicsPageMsg{Refresh: false}, cmd())
		})

		t.Run("after at least one created topic refreshes topics list", func(t *testing.T) {
			m = New(&mockCreator)

			m.Update(kadmin.TopicCreatedMsg{})

			cmd := m.Update(keys.Key(tea.KeyEsc))

			assert.Equal(t, nav.LoadTopicsPageMsg{Refresh: true}, cmd())
		})
	})

	t.Run("c-r resets form", func(t *testing.T) {
		m := New(&MockTopicCreator{})

		// topic name
		keys.UpdateKeys(m, "topicA")
		cmd := m.Update(keys.Key(tea.KeyEnter))
		m.Update(cmd())
		// partition count
		keys.UpdateKeys(m, "2")
		m.Update(cmd())
		cmd = m.Update(keys.Key(tea.KeyEnter))
		// cleanup policy
		m.Update(cmd())
		cmd = m.Update(keys.Key(tea.KeyEnter))
		// next field
		keys.UpdateKeys(m, "foo=bar")
		cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		// next field
		cmd = m.Update(cmd())
		// next group
		cmd = m.Update(cmd())

		render := m.View(ui.NewTestKontext(), ui.TestRenderer)

		assert.Contains(t, render, "Custom Topic configurations:")

		m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
		render = m.View(ui.NewTestKontext(), ui.TestRenderer)

		assert.NotContains(t, render, "Custom Topic configurations:")
	})

	t.Run("create topic", func(t *testing.T) {
		mockCreator := MockTopicCreator{
			CreateTopicFunc: func(details kadmin.TopicCreationDetails) tea.Msg {
				if details.Name == "" {
					return errors.New("topic name cannot be empty")
				}
				return CapturedTopicCreationDetails{details}
			},
		}
		m := New(&mockCreator)

		// topic name
		keys.UpdateKeys(m, "topicA")
		cmd := m.Update(keys.Key(tea.KeyEnter))
		m.Update(cmd())
		// partition count
		keys.UpdateKeys(m, "2")
		m.Update(cmd())
		cmd = m.Update(keys.Key(tea.KeyEnter))
		// cleanup policy
		m.Update(keys.Key(tea.KeyDown))
		m.Update(keys.Key(tea.KeyDown))
		cmd = m.Update(keys.Key(tea.KeyEnter))
		m.Update(cmd())
		// config - submit
		msgs := keys.Submit(m)

		var capturedDetails CapturedTopicCreationDetails
		for _, msg := range msgs {
			switch m := msg.(type) {
			case CapturedTopicCreationDetails:
				capturedDetails = m
			}
		}

		assert.NotNil(t, capturedDetails)
		assert.Equal(t, CapturedTopicCreationDetails{
			TopicCreationDetails: kadmin.TopicCreationDetails{
				"topicA",
				2,
				map[string]string{
					"cleanup.policy": "delete-compact",
				},
			},
		}, capturedDetails)
	})
}

func TestCreateTopic_Validation(t *testing.T) {
	t.Run("Validate Topic Name", func(t *testing.T) {
		t.Run("When field is empty", func(t *testing.T) {
			m := New(&MockTopicCreator{})

			cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			batchUpdate(m, cmd)

			render := m.View(&kontext.ProgramKtx{}, ui.TestRenderer)

			assert.Contains(t, render, "* Topic Name cannot be empty.")
		})
	})

	t.Run("Validate Number of Partitions", func(t *testing.T) {

		t.Run("When field is empty", func(t *testing.T) {
			m := CreateTopicSectionWithCursorAtPartitionsField()

			cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			batchUpdate(m, cmd)

			render := m.View(&kontext.ProgramKtx{}, ui.TestRenderer)

			assert.Contains(t, render, "* Number of Partitions cannot be empty.")
		})

		t.Run("When field is zero", func(t *testing.T) {
			m := CreateTopicSectionWithCursorAtPartitionsField()

			cmd := m.Update(tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{'0'},
				Alt:   false,
				Paste: false,
			})
			batchUpdate(m, cmd)
			cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			batchUpdate(m, cmd)

			render := m.View(&kontext.ProgramKtx{}, ui.TestRenderer)

			assert.Contains(t, render, "Value must be greater than zero")
		})

		t.Run("When field is negative", func(t *testing.T) {
			m := CreateTopicSectionWithCursorAtPartitionsField()

			cmd := m.Update(tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{'-'},
				Alt:   false,
				Paste: false,
			})
			batchUpdate(m, cmd)
			cmd = m.Update(tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{'1'},
				Alt:   false,
				Paste: false,
			})
			cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			batchUpdate(m, cmd)

			render := m.View(&kontext.ProgramKtx{}, ui.TestRenderer)

			assert.Contains(t, render, "Value must be greater than zero")
		})

		t.Run("When field is not a number", func(t *testing.T) {
			m := CreateTopicSectionWithCursorAtPartitionsField()

			cmd := m.Update(keys.Key('a'))
			batchUpdate(m, cmd)
			cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			batchUpdate(m, cmd)

			render := m.View(&kontext.ProgramKtx{}, ui.TestRenderer)

			assert.Contains(t, render, "'a' is not a valid numeric partition count value")
		})
	})

	t.Run("Validate configuration", func(t *testing.T) {

		t.Run("When field does not conform config=value format", func(t *testing.T) {
			m := New(&MockTopicCreator{})

			// topic name
			keys.UpdateKeys(m, "topicA")
			cmd := m.Update(keys.Key(tea.KeyEnter))
			m.Update(cmd())
			// partition count
			keys.UpdateKeys(m, "2")
			m.Update(cmd())
			cmd = m.Update(keys.Key(tea.KeyEnter))
			// cleanup policy
			m.Update(cmd())
			cmd = m.Update(keys.Key(tea.KeyEnter))
			// next field
			keys.UpdateKeys(m, "foo:bar")
			cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			render := m.View(ui.NewTestKontext(), ui.TestRenderer)

			assert.Contains(t, render, "please enter configurations in the format \"config=value\"")
		})

		t.Run("When field conforms config=value format", func(t *testing.T) {
			m := New(&MockTopicCreator{})

			// topic name
			keys.UpdateKeys(m, "topicA")
			cmd := m.Update(keys.Key(tea.KeyEnter))
			m.Update(cmd())
			// partition count
			keys.UpdateKeys(m, "2")
			m.Update(cmd())
			cmd = m.Update(keys.Key(tea.KeyEnter))
			// cleanup policy
			m.Update(cmd())
			cmd = m.Update(keys.Key(tea.KeyEnter))
			// next field
			keys.UpdateKeys(m, "foo=bar")
			cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			batchUpdate(m, cmd)

			render := m.View(ui.NewTestKontext(), ui.TestRenderer)

			assert.NotContains(t, render, "please enter configurations in the format \"config=value\"")
		})
	})
}
