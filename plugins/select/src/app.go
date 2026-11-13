package main

import (
	"context"
	"errors"
	"time"
)

// Only these methods are exposed to the trusted embedded frontend. No method
// returns a password, executes shell commands or accepts arbitrary file writes.
type App struct{ development bool }

func (a *App) DevelopmentMode() bool { return a.development }

type EnvironmentInput struct {
	Project     string     `json:"project"`
	Environment string     `json:"environment"`
	Connection  Connection `json:"connection"`
	Password    *string    `json:"password,omitempty"`
}

func (a *App) ListProjects() (map[string]Project, error) {
	c, err := loadConfig()
	// The UI only needs connection metadata; keep saved passwords write-only.
	for _, project := range c.Projects {
		for name, connection := range project.Environments {
			connection.Password = nil
			project.Environments[name] = connection
		}
	}
	return c.Projects, err
}
func (a *App) CreateProject(name string) error {
	if !validName(name) {
		return errors.New("enter a project name")
	}
	configMu.Lock()
	defer configMu.Unlock()
	c, err := loadConfig()
	if err != nil {
		return err
	}
	if _, ok := c.Projects[name]; ok {
		return errors.New("this project already exists")
	}
	c.Projects[name] = Project{Environments: map[string]Connection{}}
	return saveConfig(c)
}
func (a *App) SaveEnvironment(input EnvironmentInput) error {
	configMu.Lock()
	defer configMu.Unlock()
	c, err := loadConfig()
	if err != nil {
		return err
	}
	v, err := environmentConnection(c, input)
	if err != nil {
		return err
	}
	p := c.Projects[input.Project]
	p.Environments[input.Environment] = v
	c.Projects[input.Project] = p
	return saveConfig(c)
}

// Test the form's current values without persisting settings or credentials.
func (a *App) TestConnection(input EnvironmentInput) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}
	connection, err := environmentConnection(c, input)
	if err != nil {
		return err
	}
	password, err := storedPassword(connection)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = runQuery(ctx, connection, password, "SELECT 1 AS db_connection_check")
	return err
}

func environmentConnection(c Config, input EnvironmentInput) (Connection, error) {
	if !validName(input.Project) || !validName(input.Environment) {
		return Connection{}, errors.New("select a valid project and environment")
	}
	v, err := validateConnection(input.Connection)
	if err != nil {
		return Connection{}, err
	}
	p, ok := c.Projects[input.Project]
	if !ok {
		return Connection{}, errors.New("create the project first")
	}
	v.Password = input.Password
	if v.Password == nil {
		v.Password = p.Environments[input.Environment].Password
	}
	_, err = storedPassword(v)
	if err != nil {
		return Connection{}, err
	}
	return v, nil
}
