package cmdplx_test

import (
        "fmt"
        "io"
        "os/exec"

        "github.com/v2e4lisp/cmdplx"
)

func ExampleStart() {
        var output [2]string

        cmds := []*exec.Cmd{
                exec.Command("sh", "-c", "echo hello stderr 1>&2"),
                exec.Command("sh", "-c", "echo hello stdout"),
        }
        plx := cmdplx.New(cmds)
        plx.Start()

        for {
                select {
                case line := <-plx.Lines():
                        if err := line.Err(); err != nil {
                                if err != io.EOF {
                                        fmt.Println(err)
                                }
                                break
                        }
                        from := line.From()
                        text := line.Text()
                        output[from-1] = text
                case status := <-plx.Exit():
                        if err := status.Err(); err != nil {
                                fmt.Println(err)
                        }
                case <-plx.Done():
                        goto DONE
                }
        }
DONE:
        for _, line := range output {
                fmt.Println(line)
        }
        // Output:
        // hello stdout
        // hello stderr
}
