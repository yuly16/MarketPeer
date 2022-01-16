package wallet

// Account is application-level
type AccountInfo struct {
	Addr    string
	Balance int
	Storage map[string]interface{}
}
