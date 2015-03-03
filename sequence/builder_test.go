package sequence

import (
	"fmt"
	"testing"
	"errors"
)

func TestBuildBasic(t *testing.T) {
	out := make(chan string, 1)
	seq := SequenceOf(
		PhaseOf(func() error {
			out <- "A"
			return nil
		}).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "A1"
					return nil
				}).And(
					SequenceOf(
						PhaseOf(func() error {
							out <- "A1a"
							return nil
						}),
					).Then(
						PhaseOf(func() error {
							out <- "A1b"
							return nil
						}),
					),
				),
			),
		).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "A2"
					return nil
				}),
			),
		),
	).Then(
		PhaseOf(func() error {
			out <- "B"
			return nil
		}).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "B1"
					return nil
				}),
			),
		).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "B2"
					return nil
				}),
			),
		),
	).End(out)

	printFullSequence(seq, out)
}

func TestBuildingCompact(t *testing.T) {
	out := make(chan string, 1)
	seq := SequenceOf(
		PhaseOf(func() error {
			out <- "A"
			return nil
		}).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "A1"
					return nil
				}).And(
					FirstJust(func() error {
						out <- "A1a"
						return nil
					}).ThenJust(func() error {
						out <- "A1b"
						return nil
					}),
				),
			),
		).AndJust(func() error {
			out <- "A2"
			return nil
		}),
	).Then(
		PhaseOf(func() error {
			out <- "B"
			return nil
		}).AndJust(func() error {
			out <- "B1"
			return nil
		}).AndJust(func() error {
			out <- "B2"
			return nil
		}),
	).End(out)

	printFullSequence(seq, out)
}

func TestFailure(t *testing.T) {
	out := make(chan string, 1)
	seq := SequenceOf(
		PhaseOf(func() error {
			out <- "A"
			return nil
		}).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "A1"
					return errors.New("A failure")
				}).And(
					FirstJust(func() error {
						out <- "A1a"
						return nil
					}).ThenJust(func() error {
						out <- "A1b"
						return nil
					}),
				),
			),
		).AndJust(func() error {
			out <- "A2"
			return nil
		}),
	).Then(
		PhaseOf(func() error {
			out <- "B"
			return nil
		}).AndJust(func() error {
			out <- "B1"
			return nil
		}).AndJust(func() error {
			out <- "B2"
			return nil
		}),
	).End(out)

	printFullSequence(seq, out)
}

func TestFailureSinglePhase(t *testing.T) {
	out := make(chan string, 1)
	seq := PhaseOf(func() error {
			out <- "A"
			return nil
		}).And(
			SequenceOf(
				PhaseOf(func() error {
					out <- "A1"
					return nil
				}).And(
					FirstJust(func() error {
						out <- "A1a"
						return nil
					}).ThenJust(func() error {
						out <- "A1b"
						return nil
					}),
				),
			),
	).And(
		FirstJust(func() error {
			out <- "A2a"
			return nil
		}).ThenJust(func() error {
			out <- "A2b"
			return nil
		}).ThenJust(func() error {
			out <- "A2c"
			return errors.New("A failure")
		}).ThenJust(func() error {
			out <- "A2d"
			return nil
		}),
	).End(out)

	printFullSequence(seq, out)
}

func TestShortSinglePhase(t *testing.T) {
	out := make(chan string, 1)
	seq := PhaseOf(func() error {
		out <- "A"
		return nil
	}).AndJust(func() error {
		out <- "B"
		return nil
	}).AndJust(func() error {
		out <- "C"
		return nil
	}).End(out)

	printFullSequence(seq, out)
}

func TestSinglePhase(t *testing.T) {
	out := make(chan string, 1)
	seq := PhaseOf(func() error {
		out <- "0"
		return nil
	}).And(
		SequenceOf(
			PhaseOf(func() error {
				out <- "A"
				return nil
			}).And(
				SequenceOf(
					PhaseOf(func() error {
						out <- "A1"
						return nil
					}).And(
						SequenceOf(
							PhaseOf(func() error {
								out <- "A1a"
								return nil
							}),
						).Then(
							PhaseOf(func() error {
								out <- "A1b"
								return nil
							}),
						),
					),
				),
			).And(
				SequenceOf(
					PhaseOf(func() error {
						out <- "A2"
						return nil
					}),
				),
			),
		).Then(
			PhaseOf(func() error {
				out <- "B"
				return nil
			}).And(
				SequenceOf(
					PhaseOf(func() error {
						out <- "B1"
						return nil
					}),
				),
			).And(
				SequenceOf(
					PhaseOf(func() error {
						out <- "B2"
						return nil
					}),
				),
			),
		),
	).End(out)

	printFullSequence(seq, out)
}

func TestCompactSinglePhase(t *testing.T) {
	out := make(chan string, 1)
	seq := PhaseOf(func() error {
		out <- "0"
		return nil
	}).And(
		SequenceOf(
			PhaseOf(func() error {
				out <- "A"
				return nil
			}).And(
				SequenceOf(
					PhaseOf(func() error {
						out <- "A1"
						return nil
					}).And(
						FirstJust(func() error {
							out <- "A1a"
							return nil
						}).ThenJust(func() error {
							out <- "A1b"
							return nil
						}),
					),
				),
			).AndJust(func() error {
				out <- "A2"
				return nil
			}),
		).Then(
			PhaseOf(func() error {
				out <- "B"
				return nil
			}).AndJust(func() error {
				out <- "B1"
				return nil
			}).AndJust(func() error {
				out <- "B2"
				return nil
			}),
		),
	).End(out)

	printFullSequence(seq, out)
}


func printPhase(ph phase, prefix string, out <-chan string) bool {
	var didFail bool
	nextPrefix := fmt.Sprintf("%s|  ", prefix)
	fmt.Printf("\n%s+--", prefix)

	err := ph.main()
	print(<-out)

	if err != nil {
		didFail = true
		print("\t<------ FAILURE")
	}

	for _, seq := range ph.sequences {
		fmt.Printf("\n%s|  |", prefix)
		if !printSequence(seq.(sequence), nextPrefix, didFail, out) {
			didFail = true
		}
	}
	return !didFail
}

func printFullSequence(seq Sequence, out <-chan string) {
	printSequence(seq.sequence, "", false, out)
	print("\n\n")
}

func printSequence(seq sequence, prefix string, didFail bool, out <-chan string) bool {
	for i, ph := range seq.phases {
		if i > 0 {
			fmt.Printf("\n%s.", prefix)
			fmt.Printf("\n%s.", prefix)
		}
		if !printPhase(ph.(phase), prefix, out) {
			didFail = true
		}
		if didFail && i < len(seq.phases)-1 {
			fmt.Printf("\n%sX", prefix)
			break
		}
	}
	return !didFail
}
