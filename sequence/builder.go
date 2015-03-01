package sequence

const defaultNSequences = 4
const defaultNPhases = 4

////////////////////////////////////////////////////////////
// Phase builder
////////////////////////////////////////////////////////////

// Representation of a 'phase', a unit of computation
// made up of a main function and zero or more sequences.
//
// Call methods on this builder to combine a main function,
// and optionally sequences, into a single phase.
// Call End to recieve a Sequence object with a single phase,
// which contains all of the added units of computation.
//
// When run, a phase will spawn goroutines to run each of
// its sequences concurrently, and run its main function
// in the goroutine which triggered the run.
//
// It will then block until all of its sequences have completed.
type PhaseBuilder struct {
	main func() error
	sequences []sequence
}

// Starts building a phase with `fn` as its main function.
//
// Returns
// a phase builder with the specified main function.
func Mainly(fn func() error) PhaseBuilder {
	return PhaseBuilder{fn, make([]sequence, 0, defaultNSequences)}
}

// Adds a sequence to the to the phase.
//
// `sb` is the builder for the sequence to be added.
//
// Returns
// a copy of the reciever, but with the specified sequence added.
func (pb PhaseBuilder) Also(sb SequenceBuilder) PhaseBuilder {
	seq := sb.finish()
	return PhaseBuilder{
		main: pb.main,
		sequences: append(pb.sequences, seq),
	}
}

// Adds the function `fn`, to be run concurrently with
// other specified functions.
//
// The function will have its own sub-sequence.
//
// Returns
// a copy of the reciever, but with `fn` added.
func (pb PhaseBuilder) AlsoJust(fn func() error) PhaseBuilder {
	return pb.Also(FirstJust(fn))
}

// Finishes building so the computation may be run.
//
// Returns
// a runnable `Sequence` object containing the specified
// main function and sub-sequences.
func (pb PhaseBuilder) End(output <-chan string) Sequence {
	return SequenceOf(pb).End(output)
}

func (pb PhaseBuilder) finish() phase {
	ph := phase{}
	ph.main = pb.main
	ph.sequences = make([]runAller, len(pb.sequences))
	for i, seq := range pb.sequences {
		ph.sequences[i] = runAller(seq)
	}
	return ph
}

////////////////////////////////////////////////////////////
// Sequence builder
////////////////////////////////////////////////////////////

// Representation of a 'sequence', a unit of computation which
// contains one or more phases.
//
// Call methods on this builder to combine phases into a 
// single sequence.
// Call End to recieve a Sequence object with each of the added
// phases.
//
// When run, a sequence will run each of its phases sequentially
// in the goroutine (though these phases may spawn additional
// goroutines to do their own computation).
//
type SequenceBuilder []phase

// Starts building a sequence from a single phase.
//
// `pb` is the builder for the phase to be appended.
//
// Returns
// a builder for a new sequence containing the specified phase.
func SequenceOf(pb PhaseBuilder) SequenceBuilder {
	ph := pb.finish()
	phases := make([]phase, 1, defaultNPhases)
	phases[0] = ph
	return SequenceBuilder(phases)
}

// Starts building a sequence with a function `fn`.
//
// `fn` will be contained in its own sub-phase.
//
// Returns
// a builder for a new sequence with a single phase containing `fn`.
func FirstJust(fn func() error) SequenceBuilder {
	return SequenceOf(Mainly(fn))
}

// Appends a phase to a sequence.
//
// `pb` is the builder for the phase to be appended.
//
// Returns
// a copy of the reciever with the specified phase added.
func (sb SequenceBuilder) Then(pb PhaseBuilder) SequenceBuilder {
	ph := pb.finish()
	return SequenceBuilder(append([]phase(sb), ph))
}

// Appends a function `fn` to the sequence.
//
// `fn` will be contained in its own sub-phase.
//
// Returns
// a copy of the reciever with a phase containing `fn` added.
func (sb SequenceBuilder) ThenJust(fn func() error) SequenceBuilder {
	return sb.Then(Mainly(fn))
}

// Finishes building so the computation may be run.
//
// Returns
// a runnable `Sequence` containing the specified phases.
func (sb SequenceBuilder) End(output <-chan string) Sequence {
	return Sequence{make(chan bool, 1), sb.finish(), output}
}

func (sb SequenceBuilder) finish() sequence {
	seq := sequence{}
	seq.phases = make([]runAller, len([]phase(sb)))
	for i, ph := range []phase(sb) {
		seq.phases[i] = runAller(ph)
	}
	return seq
}
