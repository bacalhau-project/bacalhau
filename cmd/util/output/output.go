package output

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type OutputFormat string

const (
	TableFormat OutputFormat = "table"
	CSVFormat   OutputFormat = "csv"
	JSONFormat  OutputFormat = "json"
	YAMLFormat  OutputFormat = "yaml"
)

var AllFormats = append([]OutputFormat{TableFormat, CSVFormat}, NonTabularFormats...)
var NonTabularFormats = []OutputFormat{JSONFormat, YAMLFormat}

var noStyle = table.Style{
	Name:   "StyleDefault",
	Box:    table.StyleBoxDefault,
	Color:  table.ColorOptionsDefault,
	Format: table.FormatOptionsDefault,
	HTML:   table.DefaultHTMLOptions,
	Options: table.Options{
		DrawBorder:      false,
		SeparateColumns: false,
		SeparateFooter:  false,
		SeparateHeader:  false,
		SeparateRows:    false,
	},
	Title: table.TitleOptionsDefault,
}

type OutputOptions struct {
	Format     OutputFormat // The output format for the list of jobs
	Pretty     bool         // Pretty print the output
	HideHeader bool         // Hide the column headers
	NoStyle    bool         // Remove all styling from table output.
	Wide       bool         // Print full values in the table results
}

// toNonTabularOptions converts OutputOptions to NonTabularOutputOptions
func (o OutputOptions) toNonTabularOptions() NonTabularOutputOptions {
	return NonTabularOutputOptions{
		Format: o.Format,
		Pretty: o.Pretty,
	}
}

type NonTabularOutputOptions struct {
	Format OutputFormat // The output format for the list of jobs
	Pretty bool         // Pretty print the output
}

type TableColumn[T any] struct {
	table.ColumnConfig
	Value func(T) string
}

func Output[T any](cmd *cobra.Command, columns []TableColumn[T], options OutputOptions, items []T) error {
	switch options.Format {
	case TableFormat, CSVFormat:
		outputTable[T](cmd, columns, options, items)
		return nil
	default:
		return OutputNonTabular(cmd, options.toNonTabularOptions(), items)
	}
}

func OutputNonTabular[T any](cmd *cobra.Command, options NonTabularOutputOptions, items []T) error {
	switch options.Format {
	case JSONFormat:
		encoder := json.NewEncoder(cmd.OutOrStdout())
		if options.Pretty {
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(items)
	case YAMLFormat:
		b, err := yaml.Marshal(items)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(b)
		return err
	default:
		return fmt.Errorf("invalid format %q", options.Format)
	}
}

func OutputOne[T any](cmd *cobra.Command, columns []TableColumn[T], options OutputOptions, item T) error {
	switch options.Format {
	case TableFormat, CSVFormat:
		outputTable[T](cmd, columns, options, []T{item})
		return nil
	default:
		return OutputOneNonTabular(cmd, options.toNonTabularOptions(), item)
	}
}

func OutputOneNonTabular[T any](cmd *cobra.Command, options NonTabularOutputOptions, item T) error {
	switch options.Format {
	case JSONFormat:
		encoder := json.NewEncoder(cmd.OutOrStdout())
		if options.Pretty {
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(item)
	case YAMLFormat:
		b, err := yaml.Marshal(item)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(b)
		return err
	default:
		return fmt.Errorf("invalid format %q", options.Format)
	}
}

func outputTable[T any](cmd *cobra.Command, columns []TableColumn[T], options OutputOptions, items []T) {
	tw := table.NewWriter()
	tw.SetOutputMirror(cmd.OutOrStdout())

	configs := lo.Map(columns, func(c TableColumn[T], i int) table.ColumnConfig {
		config := c.ColumnConfig
		config.Number = i + 1
		if options.Wide {
			config.WidthMax = 0
			config.WidthMaxEnforcer = nil
		}
		return config
	})
	tw.SetColumnConfigs(configs)

	if !options.HideHeader {
		headers := lo.Map(columns, func(c TableColumn[T], _ int) any { return c.Name })
		tw.AppendHeader(headers)
	}

	tw.SetStyle(table.StyleColoredGreenWhiteOnBlack)
	if options.NoStyle {
		tw.SetStyle(noStyle)
	}

	for _, node := range items {
		values := lo.Map(columns, func(c TableColumn[T], _ int) any {
			return c.Value(node)
		})
		tw.AppendRow(values)
	}

	switch options.Format {
	case TableFormat:
		tw.Render()
	case CSVFormat:
		tw.RenderCSV()
	}
}
