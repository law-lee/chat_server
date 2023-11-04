package trace

import (
	"fmt"
	"io"
)

// Tracer is the interface that describes an object capable of
// tracing events throughout code.
type Tracer interface {
	Trace(...interface{})
}

//This is perfectly acceptable and valid Go code; the user
//will only ever see an object that satisfies the Tracer interface and will never even know
//about our private tracer type. Since they only interact with the interface anyway, it
//wouldn't matter if our tracer implementation exposed other methods or fields;
type tracer struct {
	out io.Writer
}

func (t *tracer) Trace(a ...interface{}) {
	fmt.Fprint(t.out, a...)
	fmt.Fprintln(t.out)
}

func New(w io.Writer) Tracer {
	return &tracer{out: w}
}

type nilTracer struct{}

func (t *nilTracer) Trace(a ...interface{}) {}

// Off creates a Tracer that will ignore calls to Trace.
func Off() Tracer {
	return &nilTracer{}
}
