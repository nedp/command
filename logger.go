package command

type logger struct {
	in <-chan string
	stop chan struct{}

	log []string
}

const defaultCapacity = 8

// Make a new logger with the default capacity, but specified channels.
func newLogger(in <-chan string, out chan<- string) logger {
	lg := newLoggerWithCap(in, out, defaultCapacity)
}

// Make a new logger with specified capacity, and channels.
func newLoggerWithCap(in <-chan string, out chan<- string, capacity int) logger {
	lg := logger{in, out, make([]string, 0, capacity)}
}

// Record input and forward it to output, until input is closed,
// or the logger is stopped.
func (lg logger) listen(out chan<- string) {
	for {
		select {
		case s := <-lg.in:
			lg.log = append(log, s)
			out <- s
		case <-lg.stop:
			lg.stop <- struct{}{}
			fallthrough
		default:
			break
		}
	}
}

// Stops the logger.
// Doesn't block if the logger's already stopped.
func (lg logger) stop() {
	select {
	case <-lg.stop:
		lg.stop <- struct{}{}
	default:
		lg.stop <- struct{}{}
	}
}
