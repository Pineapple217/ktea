package create_cluster_page

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"ktea/config"
	"ktea/kontext"
	"ktea/styles"
	"ktea/tests/keys"
	"ktea/ui"
	"testing"
)

type mockClusterRegisterer struct {
}

type capturedRegistrationDetails struct {
	config.RegistrationDetails
}

func (m mockClusterRegisterer) RegisterCluster(d config.RegistrationDetails) tea.Msg {
	return capturedRegistrationDetails{d}
}

func mockConnChecker(cluster *config.Cluster) tea.Msg {
	return cluster
}

func TestCreateClusterPage(t *testing.T) {
	ktx := kontext.ProgramKtx{
		WindowWidth:  100,
		WindowHeight: 100,
		Config: &config.Config{
			Clusters: []config.Cluster{},
		},
	}

	t.Run("Display info message when no clusters", func(t *testing.T) {
		// given
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &ktx)

		// then
		render := createEnvPage.View(&ktx, ui.TestRenderer)
		assert.Contains(t, render, "No clusters configured. Please create your first cluster!")
	})

	t.Run("Do not display info message when no clusters", func(t *testing.T) {
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &ktx)

		render := createEnvPage.View(&kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		}, ui.TestRenderer)
		assert.NotContains(t, render, "No clusters configured. Please create your first cluster!")
	})

	t.Run("Name cannot be empty", func(t *testing.T) {
		// given
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &ktx)

		// when
		createEnvPage.Update(keys.Key(tea.KeyEnter))

		// then
		render := createEnvPage.View(&ktx, ui.TestRenderer)
		assert.Contains(t, render, "name cannot be empty")
	})

	t.Run("Name must be unique", func(t *testing.T) {
		// given
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "prd",
						Color:            "#808080",
						Active:           true,
						BootstrapServers: nil,
						SASLConfig:       nil,
					},
					{
						Name:             "tst",
						Color:            "#F0F0F0",
						Active:           false,
						BootstrapServers: nil,
						SASLConfig:       nil,
					},
				},
			},
		})

		// when
		keys.UpdateKeys(createEnvPage, "prd")
		createEnvPage.Update(keys.Key(tea.KeyEnter))

		// then
		render := createEnvPage.View(&ktx, ui.TestRenderer)
		assert.Contains(t, render, "cluster prd already exists, name most be unique")
	})

	t.Run("When updating", func(t *testing.T) {
		t.Run("updates existing cluster fields", func(t *testing.T) {
			// given
			createEnvPage := NewEditForm(mockConnChecker, mockClusterRegisterer{}, &kontext.ProgramKtx{
				WindowWidth:  100,
				WindowHeight: 100,
				Config: &config.Config{
					Clusters: []config.Cluster{
						{
							Name:             "prd",
							Color:            "#808080",
							Active:           true,
							BootstrapServers: []string{":19092"},
							SASLConfig:       nil,
						},
						{
							Name:             "tst",
							Color:            "#F0F0F0",
							Active:           false,
							BootstrapServers: nil,
							SASLConfig:       nil,
						},
					},
				},
			}, &FormValues{
				Name:  "prd",
				Color: "#808080",
				Host:  ":9092",
			})

			// when
			cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
			createEnvPage.Update(cmd())
			// and: select Color
			cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
			createEnvPage.Update(cmd())
			// and: Host is entered
			for i := 0; i < len("localhost:9092"); i++ {
				createEnvPage.Update(keys.Key(tea.KeyBackspace))
			}
			keys.UpdateKeys(createEnvPage, "localhost:9091")
			cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
			// next field
			createEnvPage.Update(cmd())
			// and: auth method none is selected
			cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
			// next field
			cmd = createEnvPage.Update(cmd())
			// next group
			createEnvPage.Update(cmd())
			// and: select disabled schema registry and in doing so submitting the form
			msgs := keys.Submit(createEnvPage)

			// then
			assert.Len(t, msgs, 1)
			assert.IsType(t, &config.Cluster{}, msgs[0])
			// and
			assert.Equal(t, &config.Cluster{
				Name:             "prd",
				Color:            styles.ColorGreen,
				Active:           false,
				BootstrapServers: []string{"localhost:9091"},
				SchemaRegistry:   nil,
			}, msgs[0])
		})

		t.Run("name still has to be unique", func(t *testing.T) {
			// given
			createEnvPage := NewEditForm(mockConnChecker, mockClusterRegisterer{}, &kontext.ProgramKtx{
				WindowWidth:  100,
				WindowHeight: 100,
				Config: &config.Config{
					Clusters: []config.Cluster{
						{
							Name:             "prd",
							Color:            "#808080",
							Active:           true,
							BootstrapServers: []string{":19092"},
							SASLConfig:       nil,
						},
						{
							Name:             "tst",
							Color:            "#F0F0F0",
							Active:           false,
							BootstrapServers: nil,
							SASLConfig:       nil,
						},
					},
				},
			}, &FormValues{
				Name:  "prd",
				Color: "#808080",
				Host:  ":9092",
			})

			// when
			// delete existing prd name
			createEnvPage.Update(keys.Key(tea.KeyBackspace))
			createEnvPage.Update(keys.Key(tea.KeyBackspace))
			createEnvPage.Update(keys.Key(tea.KeyBackspace))
			// enter already existing tst name
			keys.UpdateKeys(createEnvPage, "tst")
			createEnvPage.Update(keys.Key(tea.KeyEnter))

			// then
			render := createEnvPage.View(&ktx, ui.TestRenderer)
			assert.Contains(t, render, "cluster tst already exists, name most be unique")
		})
	})

	t.Run("Host cannot be empty", func(t *testing.T) {
		// given
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		})
		// and: enter name
		keys.UpdateKeys(createEnvPage, "TST")
		cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: select Color
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())

		// when
		createEnvPage.Update(keys.Key(tea.KeyEnter))

		// then
		render := createEnvPage.View(&ktx, ui.TestRenderer)
		assert.Contains(t, render, "Host cannot be empty")
	})

	t.Run("Selecting none auth method creates cluster", func(t *testing.T) {
		// given
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		})
		// and: enter name
		keys.UpdateKeys(createEnvPage, "TST")
		cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// select Primary
		cmd = createEnvPage.Update(keys.Key(tea.KeyUp))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: select Color
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// and: Host is entered
		keys.UpdateKeys(createEnvPage, "localhost:9092")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: auth method none is selected
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		cmd = createEnvPage.Update(cmd())
		// next group
		createEnvPage.Update(cmd())
		// and: select disabled schema registry and in doing so submitting the form
		msgs := keys.Submit(createEnvPage)

		// then
		assert.Len(t, msgs, 1)
		assert.IsType(t, &config.Cluster{}, msgs[0])
		// and
		assert.Equal(t, &config.Cluster{
			Name:             "TST",
			Color:            styles.ColorRed,
			Active:           false,
			BootstrapServers: []string{"localhost:9092"},
			SchemaRegistry:   nil,
		}, msgs[0])
	})

	t.Run("Selecting SASL_SSL auth method displays username and password fields", func(t *testing.T) {
		// given
		programKtx := kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		}
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &programKtx)
		// and: enter name
		keys.UpdateKeys(createEnvPage, "TST")
		cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// select Primary
		cmd = createEnvPage.Update(keys.Key(tea.KeyUp))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: select Color
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// and: Host is entered
		keys.UpdateKeys(createEnvPage, "localhost:9092")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: auth method none is selected
		cmd = createEnvPage.Update(keys.Key(tea.KeyDown))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		createEnvPage.Update(cmd())

		// then
		render := createEnvPage.View(&programKtx, ui.TestRenderer)
		assert.Contains(t, render, "Username")
		assert.Contains(t, render, "Password")
	})

	t.Run("After selecting SASL_SSL auth method and going back to select none username and password fields are not visible anymore", func(t *testing.T) {
		// given
		programKtx := kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		}
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &programKtx)
		// and: enter name
		keys.UpdateKeys(createEnvPage, "TST")
		cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// select Primary
		cmd = createEnvPage.Update(keys.Key(tea.KeyUp))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: select Color
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// and: Host is entered
		keys.UpdateKeys(createEnvPage, "localhost:9092")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: auth method none is selected
		cmd = createEnvPage.Update(keys.Key(tea.KeyDown))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		cmd = createEnvPage.Update(cmd())
		// go back to auth method
		cmd = createEnvPage.Update(keys.Key(tea.KeyShiftTab))
		cmd = createEnvPage.Update(cmd())
		cmd = createEnvPage.Update(keys.Key(tea.KeyUp))

		// then
		render := createEnvPage.View(&programKtx, ui.TestRenderer)
		assert.NotContains(t, render, "Username")
		assert.NotContains(t, render, "Password")
	})

	t.Run("Selecting SASL_SSL auth method and filling all SASL fields creates cluster", func(t *testing.T) {
		// given
		programKtx := kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		}
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &programKtx)
		// and: enter name
		keys.UpdateKeys(createEnvPage, "TST")
		cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// select Primary
		cmd = createEnvPage.Update(keys.Key(tea.KeyUp))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: select Color
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// and: Host is entered
		keys.UpdateKeys(createEnvPage, "localhost:9092")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: auth method none is selected
		cmd = createEnvPage.Update(keys.Key(tea.KeyDown))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		cmd = createEnvPage.Update(cmd())
		// and: select SASL_SSL security protocol
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: enter SASL username
		keys.UpdateKeys(createEnvPage, "username")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: enter SASL password
		keys.UpdateKeys(createEnvPage, "password")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		cmd = createEnvPage.Update(cmd())
		// next group
		createEnvPage.Update(cmd())
		// submit
		msgs := keys.Submit(createEnvPage)

		// then
		assert.Len(t, msgs, 1)
		assert.IsType(t, &config.Cluster{}, msgs[0])
		// and
		assert.Equal(t, &config.Cluster{
			Name:             "TST",
			Color:            styles.ColorRed,
			Active:           false,
			BootstrapServers: []string{"localhost:9092"},
			SchemaRegistry:   nil,
			SASLConfig: &config.SASLConfig{
				Username:         "username",
				Password:         "password",
				SecurityProtocol: config.SSLSecurityProtocol,
			},
		}, msgs[0])
	})

	t.Run("Selecting Schema Registry Enabled opens up schema registry credentials fields", func(t *testing.T) {
		// given
		programKtx := kontext.ProgramKtx{
			WindowWidth:  100,
			WindowHeight: 100,
			Config: &config.Config{
				Clusters: []config.Cluster{
					{
						Name:             "PRD",
						BootstrapServers: []string{"localhost:9092"},
						SASLConfig:       nil,
					},
				},
			},
		}
		createEnvPage := NewForm(mockConnChecker, mockClusterRegisterer{}, &programKtx)
		// and: enter name
		keys.UpdateKeys(createEnvPage, "TST")
		cmd := createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// select Primary
		cmd = createEnvPage.Update(keys.Key(tea.KeyUp))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: select Color
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// and: Host is entered
		keys.UpdateKeys(createEnvPage, "localhost:9092")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: auth method SASL is selected
		cmd = createEnvPage.Update(keys.Key(tea.KeyDown))
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		cmd = createEnvPage.Update(cmd())
		// and: select SASL_SSL security protocol
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: enter SASL username
		keys.UpdateKeys(createEnvPage, "username")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		createEnvPage.Update(cmd())
		// and: enter SASL password
		keys.UpdateKeys(createEnvPage, "password")
		cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
		// next field
		cmd = createEnvPage.Update(cmd())
		// next group
		createEnvPage.Update(cmd())
		// select schema-registry enabled
		createEnvPage.Update(keys.Key(tea.KeyDown))

		render := createEnvPage.View(&ktx, ui.TestRenderer)
		assert.Contains(t, render, "Schema Registry URL")
		assert.Contains(t, render, "Schema Registry Username")
		assert.Contains(t, render, "Schema Registry Password")

		t.Run("Filling in the credentials and submitting creates the cluster", func(t *testing.T) {
			cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
			// next field
			createEnvPage.Update(cmd())

			// url
			keys.UpdateKeys(createEnvPage, "sr-url")
			cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
			// next field
			createEnvPage.Update(cmd())

			// username
			keys.UpdateKeys(createEnvPage, "sr-user")
			cmd = createEnvPage.Update(keys.Key(tea.KeyEnter))
			// next field
			createEnvPage.Update(cmd())

			// pwd
			keys.UpdateKeys(createEnvPage, "sr-pwd")
			msgs := keys.Submit(createEnvPage)

			// then
			assert.Len(t, msgs, 1)
			assert.IsType(t, &config.Cluster{}, msgs[0])
			// and
			assert.Equal(t, &config.Cluster{
				Name:             "TST",
				Color:            styles.ColorRed,
				Active:           false,
				BootstrapServers: []string{"localhost:9092"},
				SASLConfig: &config.SASLConfig{
					Username:         "username",
					Password:         "password",
					SecurityProtocol: config.SSLSecurityProtocol,
				},
				SchemaRegistry: &config.SchemaRegistryConfig{
					Url:      "sr-url",
					Username: "sr-user",
					Password: "sr-pwd",
				},
			}, msgs[0])
		})
	})
}
