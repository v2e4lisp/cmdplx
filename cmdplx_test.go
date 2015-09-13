package cmdplx_test

import (
        "os/exec"
        "testing"

        "github.com/v2e4lisp/cmdplx"
)

func TestStart(t *testing.T) {
        cmds := []*exec.Cmd{
                exec.Command("sh", "-c", "echo hello 1>&2"),
                exec.Command("sh", "-c", "echo world"),
                exec.Command("nosuchcommand"),
                exec.Command("sh", "-c", "exit 1"),
        }
        plx := cmdplx.New(cmds)
        plx.Start()

        var (
                exitError       *cmdplx.Status
                commandNotFound *cmdplx.Status
        )

        for {
                select {
                case line := <-plx.Lines():
                        if line.From() == 1 {
                                if text := line.Text(); text != "world" {
                                        t.Errorf("expect 'world', got %s", text)
                                }
                        }
                        if line.From() == 2 {
                                if text := line.Text(); text != "hello" {
                                        t.Errorf("expect 'hello', got %s", text)
                                }
                        }
                case status := <-plx.Started():
                        if err := status.Err(); err != nil {
                                commandNotFound = status
                        }
                case status := <-plx.Exited():
                        if err := status.Err(); err != nil {
                                exitError = status
                        }
                case <-plx.Done():
                        goto DONE
                }
        }
DONE:

        if exitError == nil || exitError.Cmd() != cmds[3] {
                t.Errorf("wrong exit status")
        }

        if commandNotFound == nil || commandNotFound.Cmd() != cmds[2] {
                t.Errorf("wrong command not found status")
        }
}
