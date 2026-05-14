package tangent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type message struct {
	Type	 string			`json:"title"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

// Used to register plugins with tangentd
type entry struct {
	Title	 string	`json:"title"`
	Subtitle string `json:"subtitle"`
	Id		 string `json:"id"`
}

// For decoding "type":"run" messages from tangentd
type runPayload struct {
	Id		 string	`json:"id"`
}

type Command struct {
	Title	 string
	Subtitle string
	handler  func()
}

type Plugin struct {
	scanner	 *bufio.Scanner
	encoder  *json.Encoder
	commands map[string]Command
	nextId   int
}

// Create a new plugin
func New() *Plugin {
	return &Plugin{
		scanner: bufio.NewScanner(os.Stdin),
		encoder: json.NewEncoder(os.Stdout),
		commands: map[string]Command{},
		nextId: 0,
	}
}

func (p *Plugin) Register(title, subtitle string, handler func()) {
	id := strconv.Itoa(p.nextId)
	p.nextId++
	p.commands[id] = Command{
		Title: title,
		Subtitle: subtitle,
		handler: handler,
	}
}

// Run announces all registered commands then blocks
// and handles incoming messages
func (plugin *Plugin) Run() error {
	for id, cmd := range plugin.commands {
		err := plugin.registerEntry(id, cmd)
		if err != nil {
			return fmt.Errorf("tangent: failed to register command %q: %w", cmd.Title, err)
		}
	}

	for plugin.scanner.Scan() {
		var msg message
		err := json.Unmarshal(plugin.scanner.Bytes(), &msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tangent: failed to unmarshal message: %v\n", err)
			continue
		}

		plugin.handle(msg)
	}

	if err := plugin.scanner.Err(); err != nil {
		return fmt.Errorf("tangent: stdin error: %w", err)
	}
	return nil
}

func (plugin *Plugin) registerEntry(id string, cmd Command) error {
	payload, err := json.Marshal(entry{
		Title: cmd.Title,
		Subtitle: cmd.Subtitle,
		Id: id,
	})
	if err != nil {
		return err
	}
	return plugin.encoder.Encode(message{Type: "RegisterEntry", Payload: payload})
}

func (p *Plugin) handle(msg message) {
	if msg.Type != "run" {
		return
	}

	var payload runPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "tangent: failed to unmarshal run payload: %v", err)
		return
	}

	cmd, ok := p.commands[payload.Id]
	if !ok {
		fmt.Fprintf(os.Stderr, "tangent: received run for unknown id %q\n", payload.Id)
	}

	cmd.handler()
}