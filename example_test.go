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
        plx := cmdplx.New(cmds)
        plx.Start()

        for {
                select {
                case line := <-plx.Lines():
                        if line == nil {
                                goto DONE
                        }
                        if err := line.Err(); err != nil {
                                if err != io.EOF {
                                        fmt.Println(err)
                                }
                                break
                        }
                        output[line.From()-1] = line.Text()
                case status := <-plx.Started():
                        if status == nil {
                                goto DONE
                        }
                        if err := status.Err(); err != nil {
                                fmt.Println(err)
                        }
                case status := <-plx.Exited():
                        if status == nil {
                                goto DONE
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
