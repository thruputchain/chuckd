package batonapi

import "fmt"

var (
	ErrNotValid          = fmt.Errorf("proposal not valid")
	ErrNothingToNominate = fmt.Errorf("nothing to nominate")
)
