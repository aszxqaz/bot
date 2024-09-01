package mexc

import "strconv"

func getPartialBookDepthStreamEndpoint(symbol string, level int) string {
	return "spot@public.limit.depth.v3.api@" + symbol + "@" + strconv.Itoa(level)
}
