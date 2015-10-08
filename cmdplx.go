package cmdplx

import (
        "bufio"
        "io"
        "os/exec"
        "sync"
)

const (
        Stdout = 1
        Stderr = 2
)

type Line struct {
        cmd *exec.Cmd // from which command
        // Error when reading from the command's output.
        // If error exists, Text and From are left unset
        err  error
        text string // line content
        from int    // from stdout or stderr
}

// Get the current command
func (l *Line) Cmd() *exec.Cmd { return l.cmd }

// Get the error occured in reading from stdout or stderr
func (l *Line) Err() error { return l.err }

// Get the current line text
func (l *Line) Text() string { return l.text }

// Check to see the current line is from Stderr or Stdout
func (l *Line) From() int { return l.from }

type Status struct {
        cmd *exec.Cmd // which command
        err error     // error
}

// Get the current command
func (s *Status) Cmd() *exec.Cmd { return s.cmd }

// Get the error
func (s *Status) Err() error { return s.err }

// Multiplex multiple commands' stdout and stderr.
type multiplexer struct {
        cmds    []*exec.Cmd  // commands to run
        lines   chan *Line   // channel to receive commands' output line by line
        exited  chan *Status // channel to receive commands' Wait() error
        started chan *Status // channel to receive commands' Start() error
        wg      *sync.WaitGroup
}

// Start all the commands
//
// Stdout and stderr will be scanned line by line and the result will be
// sent to the lines channel(unbuffered).
// Start status is sent to the started channel.
// Exit status is sent to the exited channel.
// Started and exited channel are buffered channels with the same the size of the input commands
//
// The returning channels will all get CLOSED when commands are finished
func Start(cmds []*exec.Cmd) (
        lines <-chan *Line,
        started <-chan *Status,
        exited <-chan *Status) {

        lines, started, exited = newMultiplexer(cmds).startAll()
        return
}

// Create new multiplexer
func newMultiplexer(cmds []*exec.Cmd) *multiplexer {
        plx := &multiplexer{cmds: cmds}
        plx.lines = make(chan *Line)
        plx.wg = &sync.WaitGroup{}
        count := len(plx.cmds)
        plx.exited = make(chan *Status, count)
        plx.started = make(chan *Status, count)

        return plx
}

func (plx *multiplexer) startAll() (
        lines <-chan *Line,
        started <-chan *Status,
        exited <-chan *Status) {
        for _, c := range plx.cmds {
                plx.wg.Add(1)
                go plx.start(c)
        }

        go func() {
                plx.wg.Wait()
                close(plx.lines)
                close(plx.started)
                close(plx.exited)
        }()

        return plx.lines, plx.started, plx.exited
}

// Run a command.
// Start status is sent to started channel and exit status is sent exited channel.
// Stdout and stderr will be scanned line by line and sent to lines lines channel.
func (plx *multiplexer) start(c *exec.Cmd) {
        defer plx.wg.Done()
        var (
                stdout io.ReadCloser
                stderr io.ReadCloser
        )
        status := &Status{
                cmd: c,
                err: nil,
        }

        if stdout, status.err = c.StdoutPipe(); status.err != nil {
                plx.started <- status
                return
        }
        if stderr, status.err = c.StderrPipe(); status.err != nil {
                plx.started <- status
                return
        }
        if status.err = c.Start(); status.err != nil {
                plx.started <- status
                return
        }

        outScan, errScan := bufio.NewScanner(stdout), bufio.NewScanner(stderr)
        var wg sync.WaitGroup
        wg.Add(2)
        go func() {
                defer wg.Done()
                for outScan.Scan() {
                        plx.lines <- &Line{
                                cmd:  c,
                                text: outScan.Text(),
                                from: Stdout,
                        }
                }
                if err := errScan.Err(); err != nil {
                        plx.lines <- &Line{
                                cmd: c,
                                err: err,
                        }
                }
        }()
        go func() {
                defer wg.Done()
                for errScan.Scan() {
                        plx.lines <- &Line{
                                cmd:  c,
                                text: errScan.Text(),
                                from: Stderr,
                        }
                }
                if err := outScan.Err(); err != nil {
                        plx.lines <- &Line{
                                cmd: c,
                                err: err,
                        }
                }
        }()
        wg.Wait()
        err := c.Wait()
        plx.exited <- &Status{
                cmd: c,
                err: err,
        }
}
