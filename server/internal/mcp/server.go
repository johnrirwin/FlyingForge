package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

// Server runs the MCP protocol over stdio for local clients.
type Server struct {
	protocol *Protocol
	logger   *logging.Logger
}

func NewServer(protocol *Protocol, logger *logging.Logger) *Server {
	return &Server{
		protocol: protocol,
		logger:   logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	s.logger.Info("MCP stdio server started, waiting for requests...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("read error: %w", err)
			}

			if len(line) == 0 {
				continue
			}

			response := s.protocol.HandleMessage(ctx, line)
			if response == nil {
				continue
			}

			data, err := json.Marshal(response)
			if err != nil {
				s.logger.Error("Failed to marshal MCP response", logging.WithField("error", err.Error()))
				continue
			}
			data = append(data, '\n')
			if _, err := writer.Write(data); err != nil {
				return fmt.Errorf("write error: %w", err)
			}
		}
	}
}
