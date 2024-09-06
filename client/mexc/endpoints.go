package mexc

import (
	"automata/client"
	"strconv"
)

func getPartialBookDepthStreamEndpoint(symbol client.Symbol, level int) string {
	return "spot@public.limit.depth.v3.api@" + string(symbol) + "@" + strconv.Itoa(level)
}
