package domain

type Operation struct {
	Source, Destination, Default, CodeTable string
	Mask                                    bool
}
type Mapping struct {
	ID, Name   string
	Operations []Operation
	Version    int
}
