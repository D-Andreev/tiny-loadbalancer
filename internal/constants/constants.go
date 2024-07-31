package constants

type Strategy string

const (
	RoundRobin         Strategy = "round-robin"
	Random             Strategy = "random"
	WeightedRoundRobin Strategy = "weighted-round-robin"
	IPHashing          Strategy = "ip-hashing"
	LeastConnections   Strategy = "least-connections"
)

var Strategies = []Strategy{
	RoundRobin,
	Random,
	WeightedRoundRobin,
	IPHashing,
	LeastConnections,
}
