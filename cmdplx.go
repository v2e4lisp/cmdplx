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
        // Err will be set to io.EOF if stdout and stderr
        // both received io.EOF
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
type Cmdplx struct {
        cmds    []*exec.Cmd  // commands to run
        lines   chan *Line   // channel to receive commands' output line by line
        exited  chan *Status // channel to receive commands' Wait() error
        started chan *Status // channel to receive commands' Start() error
        wg      *sync.WaitGroup
}

// Create new Cmdplx
func New(cmds []*exec.Cmd) *Cmdplx {
        plx := &Cmdplx{cmds: cmds}
        plx.lines = make(chan *Line)
        plx.wg = &sync.WaitGroup{}
        count := len(plx.cmds)
        plx.exited = make(chan *Status, count)
        plx.started = make(chan *Status, count)

        return plx
}

// Return the lines channel.
//
// Lines channel is a nonbuffered channel.
// Outputs from commands' stderr and stdout will be
// sent to this channel line by line.
//
// The line channel will not get closed by cmdplx.
func (plx *Cmdplx) Lines() <-chan *Line { return plx.lines }

// Return the exited channel.
//
// Exit channel is a bufferd channel holding all the commands Wait() error.
// Exit channel will not get closed by cmdplx.
func (plx *Cmdplx) Exited() chan *Status { return plx.exited }

// Return the started channel
//
// Start channel is a buffered channel holding all the commands Start() error.
// Start channel will not get closed by cmdplx.
func (plx *Cmdplx) Started() chan *Status { return plx.started }

// Start all the commands and wait the commands to finish in a goroutine.
//
// Stdout and stderr are sent to the lines channel.
// Exit status is sent to the exited channel.
// cmd.Start() return value will be sent to the start channel.
// When all the outputs are received and commands are finished
// the lines channel will get closed.
func (plx *Cmdplx) Start() {
        for _, c := range plx.cmds {
                err := plx.start(c)
                plx.started <- &Status{
                        cmd: c,
                        err: err,
                }
                if err != nil {
                        continue
                }

                go func(c *exec.Cmd) {
                        plx.wg.Add(1)
                        defer plx.wg.Done()
                        err := c.Wait()
                        plx.exited <- &Status{
                                cmd: c,
                                err: err,
                        }
                }(c)
        }

        go func() {
                plx.wg.Wait()
                close(plx.lines)
        }()
}

// Start a command, send its output to lines channel
func (plx *Cmdplx) start(c *exec.Cmd) error {
        stdout, err := c.StdoutPipe()
        if err != nil {
                return err
        }
        stderr, err := c.StderrPipe()
        if err != nil {
                return err
        }
        if err := c.Start(); err != nil {
                return err
        }

        outScan, errScan := bufio.NewScanner(stdout), bufio.NewScanner(stderr)
        outDone, errDone := make(chan struct{}), make(chan struct{})
        go func() {
                for outScan.Scan() {
                        plx.lines <- &Line{
                                cmd:  c,
                                text: outScan.Text(),
                                from: Stdout,
                        }
                }
                close(outDone)
        }()
        go func() {
                for errScan.Scan() {
                        plx.lines <- &Line{
                                cmd:  c,
                                text: errScan.Text(),
                                from: Stderr,
                        }
                }
                close(errDone)
        }()

        go func() {
                plx.wg.Add(1)
                defer plx.wg.Done()
                <-outDone
                <-errDone
                err := outScan.Err()
                if err == nil {
                        err = errScan.Err()
                }
                if err == nil {
                        err = io.EOF
                }
                plx.lines <- &Line{
                        cmd: c,
                        err: err,
                }
        }()

        return nil
}
