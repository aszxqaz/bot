package client

import (
	"fmt"
	"strings"
)

type QueryBuilder struct {
	sb      *strings.Builder
	isEmpty bool
}

func NewQueryBuilder() *QueryBuilder {
	q := &QueryBuilder{}
	q.isEmpty = true
	q.sb = &strings.Builder{}
	return q
}

func (q *QueryBuilder) Add(param string, value any) {
	amp := ""
	if q.isEmpty {
		q.isEmpty = false
	} else {
		amp = "&"
	}
	q.sb.WriteString(amp + param + "=" + fmt.Sprintf("%v", value))
}

func (q *QueryBuilder) String() string {
	return q.sb.String()
}
