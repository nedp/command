package sequence

import (
	"fmt"
	"testing"
	"errors"
)

func TestBuildBasic(t *testing.T) {
	seq := SequenceOf(
		Mainly(func() error {
			print("A")
			return nil
		}).Also(
			SequenceOf(
				Mainly(func() error {
					print("A1")
					return nil
				}).Also(
					SequenceOf(
						Mainly(func() error {
							print("A1a")
							return nil
						}),
					).Then(
						Mainly(func() error {
							print("A1b")
							return nil
						}),
					),
				),
			),
		).Also(
			SequenceOf(
				Mainly(func() error {
					print("A2")
					return nil
				}),
			),
		),
	).Then(
		Mainly(func() error {
			print("B")
			return nil
		}).Also(
			SequenceOf(
				Mainly(func() error {
					print("B1")
					return nil
				}),
			),
		).Also(
			SequenceOf(
				Mainly(func() error {
					print("B2")
					return nil
				}),
			),
		),
	).End()

	printFullSequence(seq)
}

func TestBuildingCompact(t *testing.T) {
	seq := SequenceOf(
		Mainly(func() error {
			print("A")
			return nil
		}).Also(
			SequenceOf(
				Mainly(func() error {
					print("A1")
					return nil
				}).Also(
					FirstJust(func() error {
						print("A1a")
						return nil
					}).ThenJust(func() error {
						print("A1b")
						return nil
					}),
				),
			),
		).AlsoJust(func() error {
			print("A2")
			return nil
		}),
	).Then(
		Mainly(func() error {
			print("B")
			return nil
		}).AlsoJust(func() error {
			print("B1")
			return nil
		}).AlsoJust(func() error {
			print("B2")
			return nil
		}),
	).End()

	printFullSequence(seq)
}

func TestFailure(t *testing.T) {
	seq := SequenceOf(
		Mainly(func() error {
			print("A")
			return nil
		}).Also(
			SequenceOf(
				Mainly(func() error {
					print("A1")
					return errors.New("A failure")
				}).Also(
					FirstJust(func() error {
						print("A1a")
						return nil
					}).ThenJust(func() error {
						print("A1b")
						return nil
					}),
				),
			),
		).AlsoJust(func() error {
			print("A2")
			return nil
		}),
	).Then(
		Mainly(func() error {
			print("B")
			return nil
		}).AlsoJust(func() error {
			print("B1")
			return nil
		}).AlsoJust(func() error {
			print("B2")
			return nil
		}),
	).End()

	printFullSequence(seq)
}

func TestFailureSinglePhase(t *testing.T) {
	seq := Mainly(func() error {
			print("A")
			return nil
		}).Also(
			SequenceOf(
				Mainly(func() error {
					print("A1")
					return nil
				}).Also(
					FirstJust(func() error {
						print("A1a")
						return nil
					}).ThenJust(func() error {
						print("A1b")
						return nil
					}),
				),
			),
	).Also(
		FirstJust(func() error {
			print("A2a")
			return nil
		}).ThenJust(func() error {
			print("A2b")
			return nil
		}).ThenJust(func() error {
			print("A2c")
			return errors.New("A failure")
		}).ThenJust(func() error {
			print("A2d")
			return nil
		}),
	).End()

	printFullSequence(seq)
}

func TestShortSinglePhase(t *testing.T) {
	seq := Mainly(func() error {
		print("A")
		return nil
	}).AlsoJust(func() error {
		print("B")
		return nil
	}).AlsoJust(func() error {
		print("C")
		return nil
	}).End()

	printFullSequence(seq)
}

func TestSinglePhase(t *testing.T) {
	seq := Mainly(func() error {
		print("0")
		return nil
	}).Also(
		SequenceOf(
			Mainly(func() error {
				print("A")
				return nil
			}).Also(
				SequenceOf(
					Mainly(func() error {
						print("A1")
						return nil
					}).Also(
						SequenceOf(
							Mainly(func() error {
								print("A1a")
								return nil
							}),
						).Then(
							Mainly(func() error {
								print("A1b")
								return nil
							}),
						),
					),
				),
			).Also(
				SequenceOf(
					Mainly(func() error {
						print("A2")
						return nil
					}),
				),
			),
		).Then(
			Mainly(func() error {
				print("B")
				return nil
			}).Also(
				SequenceOf(
					Mainly(func() error {
						print("B1")
						return nil
					}),
				),
			).Also(
				SequenceOf(
					Mainly(func() error {
						print("B2")
						return nil
					}),
				),
			),
		),
	).End()

	printFullSequence(seq)
}

func TestCompactSinglePhase(t *testing.T) {
	seq := Mainly(func() error {
		print("0")
		return nil
	}).Also(
		SequenceOf(
			Mainly(func() error {
				print("A")
				return nil
			}).Also(
				SequenceOf(
					Mainly(func() error {
						print("A1")
						return nil
					}).Also(
						FirstJust(func() error {
							print("A1a")
							return nil
						}).ThenJust(func() error {
							print("A1b")
							return nil
						}),
					),
				),
			).AlsoJust(func() error {
				print("A2")
				return nil
			}),
		).Then(
			Mainly(func() error {
				print("B")
				return nil
			}).AlsoJust(func() error {
				print("B1")
				return nil
			}).AlsoJust(func() error {
				print("B2")
				return nil
			}),
		),
	).End()

	printFullSequence(seq)
}


func printPhase(ph phase, prefix string) bool {
	var didFail bool
	nextPrefix := fmt.Sprintf("%s|  ", prefix)
	fmt.Printf("\n%s+--", prefix)
	if ph.main() != nil {
		didFail = true
		print("\t<------ FAILURE")
	}

	for _, seq := range ph.sequences {
		fmt.Printf("\n%s|  |", prefix)
		if !printSequence(seq.(sequence), nextPrefix, didFail) {
			didFail = true
		}
	}
	return !didFail
}

func printFullSequence(seq Sequence) {
	printSequence(seq.sequence, "", false)
	print("\n\n")
}

func printSequence(seq sequence, prefix string, didFail bool) bool {
	for i, ph := range seq.phases {
		if i > 0 {
			fmt.Printf("\n%s.", prefix)
			fmt.Printf("\n%s.", prefix)
		}
		if !printPhase(ph.(phase), prefix) {
			didFail = true
		}
		if didFail && i < len(seq.phases)-1 {
			fmt.Printf("\n%sX", prefix)
			break
		}
	}
	return !didFail
}
