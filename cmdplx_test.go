package cmdplx

import (
        "os/exec"
        "testing"
)

func TestStart0(t *testing.T) {
        cmds := []*exec.Cmd{
                exec.Command("sh", "-c", "echo hello 1>&2"),
                exec.Command("sh", "-c", "echo world"),
                exec.Command("nosuchcommand"),
                exec.Command("sh", "-c", "exit 1"),
        }
        plx := NewCmdplx(cmds)
        plx.Start()

        var (
                exitError       *Status
                commandNotFound *Status
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
                case status := <-plx.Exit():
                        if err := status.Err(); err != nil {
                                if _, ok := err.(*exec.ExitError); ok {
                                        exitError = status
                                } else {
                                        commandNotFound = status
                                }
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
