package nodes

import (
	"fmt"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewCmd() *cobra.Command {
	var showColumnGroups = []string{"labels", "capacity"}

	var outputFormat = output.OutputOptions{
		Format:     output.TableFormat,
		HideHeader: false,
		NoStyle:    false,
		Wide:       false,
	}

	nodesCmd := &cobra.Command{
		Use:    "nodes",
		Short:  "List nodes on the network",
		PreRun: util.ApplyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return nodes(cmd, showColumnGroups, outputFormat)
		},
	}

	nodesCmd.Flags().StringSliceVar(&showColumnGroups, "show", showColumnGroups,
		fmt.Sprintf("What column groups to show. Zero or more of: %q", maps.Keys(toggleColumns)))

	nodesCmd.Flags().AddFlagSet(flags.OutputFormatFlags(&outputFormat))

	return nodesCmd
}

func stringerizeEnum[T fmt.Stringer](val []T) string {
	return strings.Join(lo.Map[T, string](val, func(item T, _ int) string {
		return item.String()
	}), " ")
}

func maxLen(val []string) int {
	return lo.Max(lo.Map[string, int](val, func(item string, index int) int { return len(item) })) + 1
}

func ifComputeNode(getFromCNInfo func(*model.ComputeNodeInfo) string) func(model.NodeInfo) string {
	return func(ni model.NodeInfo) string {
		if ni.ComputeNodeInfo == nil {
			return ""
		}
		return getFromCNInfo(ni.ComputeNodeInfo)
	}
}

var alwaysColumns = []output.TableColumn[model.NodeInfo]{
	{
		ColumnConfig: table.ColumnConfig{Name: "id"},
		Value:        func(node model.NodeInfo) string { return system.GetShortID(node.PeerInfo.ID.String()) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "type"},
		Value:        func(ni model.NodeInfo) string { return ni.NodeType.String() },
	},
}

var toggleColumns = map[string][]output.TableColumn[model.NodeInfo]{
	"labels": {
		{
			ColumnConfig: table.ColumnConfig{Name: "labels", WidthMax: 50, WidthMaxEnforcer: text.WrapSoft},
			Value: func(ni model.NodeInfo) string {
				labels := lo.MapToSlice(ni.Labels, func(key, val string) string { return fmt.Sprintf("%s=%s", key, val) })
				slices.Sort(labels)
				return strings.Join(labels, " ")
			},
		},
	},
	"version": {
		{
			ColumnConfig: table.ColumnConfig{Name: "version"},
			Value: func(ni model.NodeInfo) string {
				return ni.BacalhauVersion.GitVersion
			},
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "architecture"},
			Value: func(ni model.NodeInfo) string {
				return ni.BacalhauVersion.GOARCH
			},
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "os"},
			Value: func(ni model.NodeInfo) string {
				return ni.BacalhauVersion.GOOS
			},
		},
	},
	"features": {
		{
			ColumnConfig: table.ColumnConfig{Name: "engines", WidthMax: maxLen(model.EngineNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.ExecutionEngines)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "verifiers", WidthMax: maxLen(model.VerifierNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.Verifiers)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "inputs from", WidthMax: maxLen(model.StorageSourceNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.StorageSources)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "outputs", WidthMax: maxLen(model.PublisherNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.Publishers)
			}),
		},
	},
	"capacity": {
		{
			ColumnConfig: table.ColumnConfig{Name: "cpu", WidthMax: len("1.0 / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%.1f / %.1f", cni.AvailableCapacity.CPU, cni.MaxCapacity.CPU)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "memory", WidthMax: len("10.0 GB / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%s / %s", datasize.ByteSize(cni.AvailableCapacity.Memory).HR(), datasize.ByteSize(cni.MaxCapacity.Memory).HR())
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "disk", WidthMax: len("100.0 GB / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%s / %s", datasize.ByteSize(cni.AvailableCapacity.Disk).HR(), datasize.ByteSize(cni.MaxCapacity.Disk).HR())
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "gpu", WidthMax: len("1 / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%d / %d", cni.AvailableCapacity.GPU, cni.MaxCapacity.GPU)
			}),
		},
	},
}

func nodes(cmd *cobra.Command, columnGroups []string, outputOpts output.OutputOptions) error {
	ctx := cmd.Context()

	nodes, err := util.GetWrappedAPIClient(ctx).Nodes(ctx)
	if err != nil {
		util.Fatal(cmd, errors.Wrap(err, "error listing jobs"), 1)
		return err
	}

	slices.SortFunc(nodes, func(a, b model.NodeInfo) bool { return a.PeerInfo.ID < b.PeerInfo.ID })

	columns := alwaysColumns
	for _, label := range columnGroups {
		columns = append(columns, toggleColumns[label]...)
	}

	return output.Output(cmd, columns, outputOpts, nodes)
}
