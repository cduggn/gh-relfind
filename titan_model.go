package main

var titanModel = "amazon.titan-embed-text-v1"

type Titan struct {
	model string
}

func (c Titan) Request() []byte {
	return nil
}
