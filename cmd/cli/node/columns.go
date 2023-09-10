package node

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/c2h5oh/datasize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

var alwaysColumns = []output.TableColumn[*models.NodeInfo]{
	{
		ColumnConfig: table.ColumnConfig{Name: "id"},
		Value:        func(node *models.NodeInfo) string { return system.GetShortID(node.PeerInfo.ID.String()) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "type"},
		Value:        func(ni *models.NodeInfo) string { return ni.NodeType.String() },
	},
}

var toggleColumns = map[string][]output.TableColumn[*models.NodeInfo]{
	"labels": {
		{
			ColumnConfig: table.ColumnConfig{Name: "labels", WidthMax: 50, WidthMaxEnforcer: text.WrapSoft},
			Value: func(ni *models.NodeInfo) string {
				labels := lo.MapToSlice(ni.Labels, func(key, val string) string { return fmt.Sprintf("%s=%s", key, val) })
				slices.Sort(labels)
				return strings.Join(labels, " ")
			},
		},
	},
	"version": {
		{
			ColumnConfig: table.ColumnConfig{Name: "version"},
			Value: func(ni *models.NodeInfo) string {
				return ni.BacalhauVersion.GitVersion
			},
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "architecture"},
			Value: func(ni *models.NodeInfo) string {
				return ni.BacalhauVersion.GOARCH
			},
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "os"},
			Value: func(ni *models.NodeInfo) string {
				return ni.BacalhauVersion.GOOS
			},
		},
	},
	"features": {
		{
			ColumnConfig: table.ColumnConfig{Name: "engines", WidthMax: maxLen(model.EngineNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return strings.Join(cni.ExecutionEngines, " ")
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "inputs from", WidthMax: maxLen(model.StorageSourceNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return strings.Join(cni.StorageSources, " ")
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "outputs", WidthMax: maxLen(model.PublisherNames()), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return strings.Join(cni.Publishers, " ")
			}),
		},
	},
	"capacity": {
		{
			ColumnConfig: table.ColumnConfig{Name: "cpu", WidthMax: len("1.0 / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return fmt.Sprintf("%.1f / %.1f", cni.AvailableCapacity.CPU, cni.MaxCapacity.CPU)
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "memory", WidthMax: len("10.0 GB / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return fmt.Sprintf("%s / %s", datasize.ByteSize(cni.AvailableCapacity.Memory).HR(), datasize.ByteSize(cni.MaxCapacity.Memory).HR())
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "disk", WidthMax: len("100.0 GB / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return fmt.Sprintf("%s / %s", datasize.ByteSize(cni.AvailableCapacity.Disk).HR(), datasize.ByteSize(cni.MaxCapacity.Disk).HR())
			}),
		},
		{
			ColumnConfig: table.ColumnConfig{Name: "gpu", WidthMax: len("1 / "), WidthMaxEnforcer: text.WrapSoft},
			Value: ifComputeNode(func(cni *models.ComputeNodeInfo) string {
				return fmt.Sprintf("%d / %d", cni.AvailableCapacity.GPU, cni.MaxCapacity.GPU)
			}),
		},
	},
}

func maxLen(val []string) int {
	return lo.Max(lo.Map[string, int](val, func(item string, index int) int { return len(item) })) + 1
}

func ifComputeNode(getFromCNInfo func(*models.ComputeNodeInfo) string) func(*models.NodeInfo) string {
	return func(ni *models.NodeInfo) string {
		if ni.ComputeNodeInfo == nil {
			return ""
		}
		return getFromCNInfo(ni.ComputeNodeInfo)
	}
}
