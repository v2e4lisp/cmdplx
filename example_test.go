package cmdplx_test

import (
        "fmt"
        "io"
        "os/exec"

        "github.com/v2e4lisp/cmdplx"
)

func Example() {
        var output [2]string

        cmds := []*exec.Cmd{
                exec.Command("sh", "-c", "echo stderr 1>&2"),
                exec.Command("sh", "-c", "echo stdout"),
        }
        closed := 0
        lines, started, exited := cmdplx.Start(cmds)

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
                        if err := line.Err(); err != nil {
                                if err != io.EOF {
                                        fmt.Println(err)
                                }
                                break
                        }
                        output[line.From()-1] = line.Text()
                case status := <-started:
                        if status == nil {
                                started = nil
                                if closed++; closed == 3 {
                                        goto DONE
                                }
                                break
                        }
                        if err := status.Err(); err != nil {
                                fmt.Println(err)
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
                                fmt.Println(err)
                        }
                }
        }
DONE:
        fmt.Println(output)
        // Output:
        // [stdout stderr]
}
