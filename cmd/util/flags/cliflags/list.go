package cliflags

import (
	"strings"

	"github.com/spf13/pflag"
)

type ListOptions struct {
	Limit         uint32
	NextToken     string
	OrderBy       string
	OrderByFields []string
	Reverse       bool
}

func ListFlags(options *ListOptions) *pflag.FlagSet {
	flagset := pflag.NewFlagSet("List settings", pflag.ContinueOnError)
	flagset.Uint32Var(&options.Limit, "limit", options.Limit, "Limit the number of results returned")
	flagset.StringVar(&options.NextToken, "next-token", options.NextToken, "Next token to use for pagination")

	orderByUsage := "Order results by a field"
	if len(options.OrderByFields) > 0 {
		orderByUsage += ". Valid fields are: " + strings.Join(options.OrderByFields, ", ")
	}
	flagset.StringVar(&options.OrderBy, "order-by", options.OrderBy, orderByUsage)
	flagset.BoolVar(&options.Reverse, "order-reversed", options.Reverse, "Reverse the order of the results")
	return flagset
}
