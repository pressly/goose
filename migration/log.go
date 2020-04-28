package migration

import (
	"fmt"
	std "github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"io"
	"os"
	"sync"
)

var log std.Logger

func init() {
	std.SetHandler(cli.New(os.Stdout))
	log = std.Logger{
		Handler: GanderHandler{
			mu:      sync.Mutex{},
			Writer:  os.Stdout,
			Padding: 0,
		},
		Level:   std.DebugLevel,
	}
}

type GanderHandler struct {
	mu      sync.Mutex
	Writer  io.Writer
	Padding int
}

func (h GanderHandler) HandleLog(e *std.Entry) error {
	color := cli.Colors[e.Level]
	names := e.Fields.Names()

	h.mu.Lock()
	defer h.mu.Unlock()

	color.Fprintf(h.Writer, "%-25s", e.Message)

	for _, name := range names {
		if name == "source" {
			continue
		}
		fmt.Fprintf(h.Writer, " %s=%v", color.Sprint(name), e.Fields.Get(name))
	}

	fmt.Fprintln(h.Writer)

	return nil
}
