package sdk

import "fmt"

var (
	ErrNoExecs           = fmt.Errorf("no executors")
	ErrNotValid          = fmt.Errorf("proposal not valid")
	ErrNothingToNominate = fmt.Errorf("nothing to nominate")
)
