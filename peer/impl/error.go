package impl

import (
	"errors"
	"fmt"
)

var OnProposing error = errors.New("consensus is now proposing")

type SenderCallbackError struct {
	err error
}

func (sce *SenderCallbackError) Error() string {
	return fmt.Sprintf("SenderCallbackError: %s", sce.err)
}
