package command

type logger struct {
	in <-chan string
	stopCh chan struct{}

	log []string
}

const defaultCapacity = 8

// Make a new logger with the default capacity, but specified channels.
func newLogger(in <-chan string) logger {
	return newLoggerWithCap(in, len(in))
}

// Make a new logger with specified capacity, and channels.
func newLoggerWithCap(in <-chan string, capacity int) logger {
	return logger{in, make(chan struct{}, 1), make([]string, 0, capacity)}
}

// Record input and forward it to output, until input is closed,
// or the logger is stopped.
func (lg *logger) listen(out chan<- string) {
	for {
		select {
		case s := <-lg.in:
			lg.log = append(lg.log, s)
			out <- s
		case <-lg.stopCh:
			lg.stopCh <- struct{}{}
			break
		default:
			break
		}
	}
}

// Stops the logger.
// Doesn't block if the logger's already stopped.
func (lg logger) stop() {
	select {
	case <-lg.stopCh:
		lg.stopCh <- struct{}{}
	default:
		lg.stopCh <- struct{}{}
	}
}
