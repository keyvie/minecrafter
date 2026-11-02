package modules

import (
	"fmt"

	"github.com/keyvie/minecrafter/internal/core"
)

func AddCommand[S any](container *core.Container[S], command *core.ApplicationCommand[S]) error {
	key := command.Key()
	if _, exists := container.ApplicationCommands[key]; exists {
		return fmt.Errorf("command %s already exists as type %d", key.Name, key.Type)
	}
	container.ApplicationCommands[key] = command
	return nil
}

func AddCommands[S any](container *core.Container[S], commands ...*core.ApplicationCommand[S]) error {
	for _, command := range commands {
		if err := AddCommand(container, command); err != nil {
			return err
		}
	}
	return nil
}

func RemoveCommand[S any](container *core.Container[S], key core.ApplicationCommandKey) error {
	if _, exists := container.ApplicationCommands[key]; !exists {
		return fmt.Errorf("command %s does not exist as type %d", key.Name, key.Type)
	}
	delete(container.ApplicationCommands, key)
	return nil
}

func GetCommand[S any](container *core.Container[S], key core.ApplicationCommandKey) (*core.ApplicationCommand[S], error) {
	if command, exists := container.ApplicationCommands[key]; exists {
		return command, nil
	}
	return nil, fmt.Errorf("command %s does not exist as type %d", key.Name, key.Type)
}

func ListCommands[S any](container *core.Container[S]) []*core.ApplicationCommand[S] {
	commands := make([]*core.ApplicationCommand[S], 0, len(container.ApplicationCommands))
	for _, command := range container.ApplicationCommands {
		commands = append(commands, command)
	}
	return commands
}
