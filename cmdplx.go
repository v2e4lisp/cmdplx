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

// get the current command
func (l *Line) Cmd() *exec.Cmd { return l.cmd }

// get the error occured in reading from stdout or stderr
func (l *Line) Err() error { return l.err }

// get the current line text
func (l *Line) Text() string { return l.text }

// the line is from stderr or stdout
func (l *Line) From() int { return l.from }

type Status struct {
        cmd *exec.Cmd // which command
        err error     // exit error
}

// get the current command
func (s *Status) Cmd() *exec.Cmd { return s.cmd }

// get the command exit status
func (s *Status) Err() error { return s.err }

// multiplex multiple commands' stdout and stderr
type Cmdplx struct {
        cmds  []*exec.Cmd   // commands to run
        lines chan *Line    // channel to receive commands' output line by line
        done  chan struct{} // channel to inform all the commands are finished and all the outputs are received
        exit  chan *Status  // channel to receive commands' exit status
        wg    *sync.WaitGroup
}

// Create new Cmdplx
func NewCmdplx(cmds []*exec.Cmd) *Cmdplx {
        plx := &Cmdplx{cmds: cmds}
        plx.lines = make(chan *Line)
        plx.done = make(chan struct{})
        plx.wg = &sync.WaitGroup{}
        count := len(plx.cmds)
        plx.exit = make(chan *Status, count)

        return plx
}

// return the lines channel
func (plx *Cmdplx) Lines() chan *Line { return plx.lines }

// return the done channel
func (plx *Cmdplx) Done() chan struct{} { return plx.done }

// return the exit channel
func (plx *Cmdplx) Exit() chan *Status { return plx.exit }

// Start all the commands and wait them to finish.
//
// Stdout and stderr are sent to the lines channel.
// Exit status is sent to the exit channel. If a command failed to start,
// its error will also be sent to the exit channel.
// When all the outputs are received and commands are finished
// the done channel will be closed.
func (plx *Cmdplx) Start() {
        for _, c := range plx.cmds {
                if err := plx.start(c); err != nil {
                        plx.exit <- &Status{
                                cmd: c,
                                err: err,
                        }
                        continue
                }

                go func(c *exec.Cmd) {
                        plx.wg.Add(1)
                        defer plx.wg.Done()
                        err := c.Wait()
                        plx.exit <- &Status{
                                cmd: c,
                                err: err,
                        }
                }(c)
        }

        go func() {
                plx.wg.Wait()
                close(plx.done)
        }()
}

// Start a command, send its output to lines channel
// and wait the command to finish
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
