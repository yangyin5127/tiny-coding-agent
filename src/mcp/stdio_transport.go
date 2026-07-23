package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
}

func NewStdioTransport(ctx context.Context, config *MCPServerConfig) (*StdioTransport, error) {

	cmd := exec.CommandContext(ctx, config.Command, config.Args...)

	if len(config.Env) > 0 {
		cmd.Env = os.Environ()

		// Set environment variables for the command.
		for k, v := range config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

func (s *StdioTransport) Close() error {
	if s.cmd != nil && s.cmd.Process != nil {
		err := s.cmd.Process.Kill()
		return err
	}
	if s.stdin != nil {
		err := s.stdin.Close()
		return err
	}
	return nil
}

func (s *StdioTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	_, err = s.stdin.Write(data)
	if err != nil {
		return nil, err
	}

	line, err := s.stdout.ReadString('\n')
	if err != nil {
		return nil, err
	}
	var resp Response
	err = json.Unmarshal([]byte(line), &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *StdioTransport) Notify(ctx context.Context, req *Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = s.stdin.Write(data)
	return err
}
