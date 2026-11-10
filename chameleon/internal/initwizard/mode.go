package initwizard

import (
	"fmt"
)

// MaxPasswordAttempts is how many times the password question is repeated.
const MaxPasswordAttempts = 3

// ModeOffer carries what OfferStandardMode needs. The functions hide the
// vault, the journal and the terminal, so the flow can be tested.
type ModeOffer struct {
	Prompter     *Prompter
	ReadPassword func(prompt string) (string, error)
	MinLength    int
	SetPassword  func(password string) error
	SetStandard  func() error
}

// OfferStandardMode asks, after the first migrate --apply, whether to set a
// password and switch from readonly to standard. It returns true only when the
// mode was changed. Declining, or failing to give a valid password three
// times, leaves readonly and returns false without an error.
func OfferStandardMode(o ModeOffer) (bool, error) {
	out := o.Prompter.Out

	if !o.Prompter.Confirm("Set a password and switch to standard mode now?", false) {
		return false, nil
	}

	for attempt := 1; attempt <= MaxPasswordAttempts; attempt++ {
		first, err := o.ReadPassword(fmt.Sprintf("Choose mode password (min %d chars): ", o.MinLength))
		if err != nil {
			return false, err
		}
		second, err := o.ReadPassword("Repeat mode password: ")
		if err != nil {
			return false, err
		}

		switch {
		case len(first) < o.MinLength:
			fmt.Fprintf(out, "Password too short (minimum %d characters).\n", o.MinLength)
			continue
		case first != second:
			fmt.Fprintln(out, "The passwords do not match.")
			continue
		}

		if err := o.SetPassword(first); err != nil {
			return false, err
		}
		if err := o.SetStandard(); err != nil {
			return false, fmt.Errorf("the password was set, but the mode could not be changed: %w", err)
		}
		return true, nil
	}

	fmt.Fprintln(out, "Staying in readonly mode.")
	return false, nil
}
