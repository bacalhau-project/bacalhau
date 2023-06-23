package nodes

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/c2h5oh/datasize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var showColumnGroups = []string{"labels", "capacity"}

func NewCmd() *cobra.Command {
	nodesCmd := &cobra.Command{
		Use:    "nodes",
		Short:  "List nodes on the network",
		PreRun: util.ApplyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return nodes(cmd)
		},
	}

	nodesCmd.Flags().StringSliceVar(&showColumnGroups, "show", showColumnGroups,
		fmt.Sprintf("What column groups to show. Zero or more of: %q", maps.Keys(toggleColumns)))

	return nodesCmd
}

func stringerizeEnum[T fmt.Stringer](val []T) string {
	return strings.Join(lo.Map[T, string](val, func(item T, _ int) string {
		return item.String()
	}), ", ")
}

func ifComputeNode(getFromCNInfo func(*model.ComputeNodeInfo) string) func(model.NodeInfo) string {
	return func(ni model.NodeInfo) string {
		if ni.ComputeNodeInfo == nil {
			return ""
		}
		return getFromCNInfo(ni.ComputeNodeInfo)
	}
}

type tableColumn[T any] struct {
	table.ColumnConfig
	Value func(T) string
}

var alwaysColumns = []tableColumn[model.NodeInfo]{
	{
		ColumnConfig: table.ColumnConfig{Name: "id"},
		Value:        func(node model.NodeInfo) string { return system.GetShortID(node.PeerInfo.ID.String()) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "type"},
		Value:        func(ni model.NodeInfo) string { return ni.NodeType.String() },
	},
}

var toggleColumns = map[string][]tableColumn[model.NodeInfo]{
	"labels": {
		{
			ColumnConfig: table.ColumnConfig{Name: "labels"},
			Value: func(ni model.NodeInfo) string {
				labels := lo.MapToSlice(ni.Labels, func(key, val string) string { return fmt.Sprintf("%s=%s", key, val) })
				slices.Sort(labels)
				return strings.Join(labels, ", ")
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
			ColumnConfig: table.ColumnConfig{Name: "engines"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.ExecutionEngines)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "verifiers"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.Verifiers)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "inputs from"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.StorageSources)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "outputs to"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return stringerizeEnum(cni.Publishers)
			}),
		},
	},
	"capacity": {
		{
			ColumnConfig: table.ColumnConfig{Name: "cpu"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%.1f / %.1f", cni.AvailableCapacity.CPU, cni.MaxCapacity.CPU)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "memory"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%s / %s", datasize.ByteSize(cni.AvailableCapacity.Memory).HR(), datasize.ByteSize(cni.MaxCapacity.Memory).HR())
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "disk"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%s / %s", datasize.ByteSize(cni.AvailableCapacity.Disk).HR(), datasize.ByteSize(cni.MaxCapacity.Disk).HR())
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "gpu"},
			Value: ifComputeNode(func(cni *model.ComputeNodeInfo) string {
				return fmt.Sprintf("%d / %d", cni.AvailableCapacity.GPU, cni.MaxCapacity.GPU)
			}),
		},
	},
}

func nodes(cmd *cobra.Command) error {
	ctx := cmd.Context()

	nodes, err := util.GetAPIClient(ctx).Nodes(ctx)
	if err != nil {
		util.Fatal(cmd, errors.Wrap(err, "error listing jobs"), 1)
		return err
	}

	columns := alwaysColumns
	for _, label := range showColumnGroups {
		columns = append(columns, toggleColumns[label]...)
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(cmd.OutOrStderr())

	configs := lo.Map(columns, func(c tableColumn[model.NodeInfo], _ int) table.ColumnConfig { return c.ColumnConfig })
	tw.SetColumnConfigs(configs)

	headers := lo.Map(columns, func(c tableColumn[model.NodeInfo], _ int) any { return c.Name })
	tw.AppendHeader(headers)

	for _, node := range nodes {
		values := lo.Map(columns, func(c tableColumn[model.NodeInfo], _ int) any {
			return c.Value(node)
		})
		tw.AppendRow(values)
	}

	tw.Render()

	return nil
}
