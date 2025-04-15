package test

type testAddress struct {
	Address string
}

func newTestAddress(addr string) testAddress {
	return testAddress{Address: addr}
}

func (x testAddress) Network() string {
	return "testnet"
}

func (x testAddress) String() string {
	return x.Address
}
