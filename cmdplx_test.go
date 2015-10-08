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
        lines, started, exited := cmdplx.Start(cmds)
        closed := 0
        var (
                exitError       *cmdplx.Status
                commandNotFound *cmdplx.Status
        )

        for {
                select {
                case line := <-lines:
                        if line == nil {
                                lines = nil
                                if closed++; closed == 3 {
                                        goto DONE
                                }
                                break
                        }
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
                case status := <-started:
                        if status == nil {
                                started = nil
                                if closed++; closed == 3 {
                                        goto DONE
                                }
                                break
                        }
                        if err := status.Err(); err != nil {
                                commandNotFound = status
                        }
                case status := <-exited:
                        if status == nil {
                                exited = nil
                                if closed++; closed == 3 {
                                        goto DONE
                                }
                                break
                        }
                        if err := status.Err(); err != nil {
                                exitError = status
                        }
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
