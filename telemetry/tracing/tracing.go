package tracing

type Tracer interface {
	Start() (End, error)
}

type End func()
