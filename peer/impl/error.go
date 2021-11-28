package impl

import "fmt"

type SenderCallbackError struct {
	err error
}

func (sce *SenderCallbackError) Error() string {
	return fmt.Sprintf("SenderCallbackError: %s", sce.err)
}
