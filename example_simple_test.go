package cmdplx_test

import (
        "fmt"
        "io"
        "os/exec"

        "github.com/v2e4lisp/cmdplx"
)

func ExampleSimple() {
        var output [2]string

        plx := cmdplx.New([]*exec.Cmd{
                exec.Command("sh", "-c", "echo stderr 1>&2"),
                exec.Command("sh", "-c", "echo stdout"),
        })
        plx.Start()

        for line := range plx.Lines() {
                if err := line.Err(); err != nil {
                        if err != io.EOF {
                                fmt.Println(err)
                        }
                        continue
                }
                output[line.From()-1] = line.Text()
        }

        fmt.Println(output)
        // Output:
        // [stdout stderr]
}
