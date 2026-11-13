package main

import (
	"errors"
	"strings"
)

func changeConfiguration(change func(*Config) error) error {
	configMu.Lock()
	defer configMu.Unlock()
	config, err := loadConfig()
	if err != nil {
		return err
	}
	if err := change(&config); err != nil {
		return err
	}
	return saveConfig(config)
}

func (a *App) RenameProject(name, newName string) error {
	newName = strings.TrimSpace(newName)
	if !validName(newName) {
		return errors.New("enter a project name")
	}
	return changeConfiguration(func(config *Config) error {
		project, ok := config.Projects[name]
		if !ok {
			return errors.New("project is not configured")
		}
		if name != newName {
			if _, exists := config.Projects[newName]; exists {
				return errors.New("this project already exists")
			}
			config.Projects[newName] = project
			delete(config.Projects, name)
		}
		return nil
	})
}

func (a *App) DeleteProject(name string) error {
	return changeConfiguration(func(config *Config) error {
		if _, ok := config.Projects[name]; !ok {
			return errors.New("project is not configured")
		}
		delete(config.Projects, name)
		return nil
	})
}

func (a *App) SetProjectColor(name, color string) error {
	if !validProjectColor(color) {
		return errors.New("choose one of the project colors")
	}
	return changeConfiguration(func(config *Config) error {
		project, ok := config.Projects[name]
		if !ok {
			return errors.New("project is not configured")
		}
		project.Color = color
		config.Projects[name] = project
		return nil
	})
}

func (a *App) RenameEnvironment(projectName, name, newName string) error {
	newName = strings.TrimSpace(newName)
	if !validName(newName) {
		return errors.New("enter an environment name")
	}
	return changeConfiguration(func(config *Config) error {
		project, ok := config.Projects[projectName]
		if !ok {
			return errors.New("project is not configured")
		}
		connection, ok := project.Environments[name]
		if !ok {
			return errors.New("environment is not configured")
		}
		if name != newName {
			if _, exists := project.Environments[newName]; exists {
				return errors.New("this environment already exists")
			}
			project.Environments[newName] = connection
			delete(project.Environments, name)
		}
		return nil
	})
}

func (a *App) DeleteEnvironment(projectName, name string) error {
	return changeConfiguration(func(config *Config) error {
		project, ok := config.Projects[projectName]
		if !ok {
			return errors.New("project is not configured")
		}
		if _, ok := project.Environments[name]; !ok {
			return errors.New("environment is not configured")
		}
		delete(project.Environments, name)
		return nil
	})
}

func (a *App) SetEnvironmentColor(projectName, name, color string) error {
	if !validEnvironmentColor(color) {
		return errors.New("choose one of the environment colors")
	}
	return changeConfiguration(func(config *Config) error {
		project, ok := config.Projects[projectName]
		if !ok {
			return errors.New("project is not configured")
		}
		env, ok := project.Environments[name]
		if !ok {
			return errors.New("environment does not exist")
		}
		env.Color = color
		project.Environments[name] = env
		return nil
	})
}
