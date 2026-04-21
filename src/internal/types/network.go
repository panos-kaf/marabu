package types

type Origin string

const (
	Inbound  Origin = "inbound"
	Outbound Origin = "outbound"
)

type Direction string

const (
	Sent Direction = "sent"
	Recv Direction = "recv"
)
