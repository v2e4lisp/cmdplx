package cmdplx_test

import (
        "fmt"
        "os/exec"

        "github.com/v2e4lisp/cmdplx"
)

func Example_simple() {
        var output [2]string

        plx := cmdplx.New([]*exec.Cmd{
                exec.Command("sh", "-c", "echo stderr 1>&2"),
                exec.Command("sh", "-c", "echo stdout"),
        })
        lines, _, _ := plx.Start()

        for line := range lines {
                if err := line.Err(); err == nil {
                        output[line.From()-1] = line.Text()
                }
        }

        fmt.Println(output)
        // Output:
        // [stdout stderr]
}
