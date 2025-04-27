package analytics

import (
	"math"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	// NodeInfoEventType is the event type for node information
	NodeInfoEventType = "bacalhau.compute_nodes_v1.info"

	// maxEnginesToReport is the maximum number of execution engines to report in the event
	maxEnginesToReport = 10
)

// NewNodeInfosEvent creates a new analytics event from a collection of node states
func NewNodeInfosEvent(nodes []models.NodeState) Event {
	// Filter and get compute nodes
	nodes = filterComputeNodes(nodes)

	// Return early if no compute nodes
	if len(nodes) == 0 {
		return NoopEvent
	}

	properties := make(EventProperties)

	// Add basic node metrics
	properties["total_nodes"] = len(nodes)

	// Process node states - adds nodes_by_state_* properties
	populateNodeStateProperties(nodes, properties)

	// Process resources - adds resources_total_* and resources_available_* properties
	totalResources, _ := populateResourceProperties(nodes, properties)

	// Process resource statistics - adds resource_stats_* properties
	populateResourceStatsProperties(nodes, properties)

	// Process capabilities - adds capabilities_* properties
	populateCapabilitiesProperties(nodes, properties)

	// Process utilization - adds utilization_* properties
	populateUtilizationProperties(nodes, totalResources, properties)

	return NewEvent(NodeInfoEventType, properties)
}

// filterComputeNodes filters the list of nodes to only include compute nodes
func filterComputeNodes(nodes []models.NodeState) []models.NodeState {
	var computeNodes []models.NodeState

	for _, node := range nodes {
		if node.Info.IsComputeNode() {
			computeNodes = append(computeNodes, node)
		}
	}

	return computeNodes
}

// populateNodeStateProperties counts nodes by state and adds them to properties
// Properties populated:
//   - nodes_by_state_<status>: count of nodes in each connection status
func populateNodeStateProperties(nodes []models.NodeState, properties EventProperties) {
	nodesByState := make(map[string]int)

	for _, node := range nodes {
		nodesByState[strings.ToLower(node.ConnectionState.Status.String())]++
	}

	// Add node state counts to properties
	for state, count := range nodesByState {
		properties["nodes_by_state_"+state] = count
	}
}

// populateResourceProperties calculates total and available resources and adds them to properties
// Properties populated:
//   - resources_total_cpu: total CPU capacity across all nodes
//   - resources_total_memory: total memory capacity across all nodes
//   - resources_total_disk: total disk capacity across all nodes
//   - resources_total_gpu: total GPU capacity across all nodes
//   - resources_available_cpu: available CPU capacity across all nodes
//   - resources_available_memory: available memory capacity across all nodes
//   - resources_available_disk: available disk capacity across all nodes
//   - resources_available_gpu: available GPU capacity across all nodes
func populateResourceProperties(nodes []models.NodeState, properties EventProperties) (models.Resources, models.Resources) {
	var totalResources models.Resources
	var availableResources models.Resources

	for _, node := range nodes {
		// Track total resources (MaxCapacity)
		totalResources = *totalResources.Add(node.Info.ComputeNodeInfo.MaxCapacity)

		// Track available resources
		availableResources = *availableResources.Add(node.Info.ComputeNodeInfo.AvailableCapacity)
	}

	// Add resource information directly to properties
	properties["resources_total_cpu"] = totalResources.CPU
	properties["resources_total_memory"] = totalResources.Memory
	properties["resources_total_disk"] = totalResources.Disk
	properties["resources_total_gpu"] = totalResources.GPU

	properties["resources_available_cpu"] = availableResources.CPU
	properties["resources_available_memory"] = availableResources.Memory
	properties["resources_available_disk"] = availableResources.Disk
	properties["resources_available_gpu"] = availableResources.GPU

	return totalResources, availableResources
}

// populateResourceStatsProperties computes statistics for node resources and adds them to properties
// Properties populated:
//   - resource_stats_min_cpu: minimum CPU capacity of any node
//   - resource_stats_min_memory: minimum memory capacity of any node
//   - resource_stats_min_disk: minimum disk capacity of any node
//   - resource_stats_min_gpu: minimum GPU capacity of any node
//   - resource_stats_max_cpu: maximum CPU capacity of any node
//   - resource_stats_max_memory: maximum memory capacity of any node
//   - resource_stats_max_disk: maximum disk capacity of any node
//   - resource_stats_max_gpu: maximum GPU capacity of any node
//   - resource_stats_avg_cpu: average CPU capacity across nodes
//   - resource_stats_avg_memory: average memory capacity across nodes
//   - resource_stats_avg_disk: average disk capacity across nodes
//   - resource_stats_avg_gpu: average GPU capacity across nodes
//   - resource_stats_std_dev_cpu: std deviation of CPU capacity across nodes
//   - resource_stats_std_dev_memory: std deviation of memory capacity across nodes
//   - resource_stats_std_dev_disk: std deviation of disk capacity across nodes
//   - resource_stats_std_dev_gpu: std deviation of GPU capacity across nodes
func populateResourceStatsProperties(nodes []models.NodeState, properties EventProperties) {
	var minResources models.Resources
	var maxResources models.Resources
	var sumSquaredResources models.Resources
	var sumResources models.Resources
	nodesWithResources := 0

	// Gather resource information
	for i, node := range nodes {
		if node.Info.ComputeNodeInfo.AvailableCapacity.IsZero() {
			continue
		}

		resources := node.Info.ComputeNodeInfo.AvailableCapacity
		nodesWithResources++

		// Initialize min/max for first node with resources
		if i == 0 || (minResources.IsZero()) {
			minResources = resources
			maxResources = resources
		} else {
			// Update min values
			if resources.CPU < minResources.CPU {
				minResources.CPU = resources.CPU
			}
			if resources.Memory < minResources.Memory {
				minResources.Memory = resources.Memory
			}
			if resources.Disk < minResources.Disk {
				minResources.Disk = resources.Disk
			}
			if resources.GPU < minResources.GPU {
				minResources.GPU = resources.GPU
			}

			// Update max values
			if resources.CPU > maxResources.CPU {
				maxResources.CPU = resources.CPU
			}
			if resources.Memory > maxResources.Memory {
				maxResources.Memory = resources.Memory
			}
			if resources.Disk > maxResources.Disk {
				maxResources.Disk = resources.Disk
			}
			if resources.GPU > maxResources.GPU {
				maxResources.GPU = resources.GPU
			}
		}

		// Track for average and standard deviation calculation
		sumResources = *sumResources.Add(resources)
		sumSquaredResources = *sumSquaredResources.Add(models.Resources{
			CPU:    resources.CPU * resources.CPU,
			Memory: resources.Memory * resources.Memory,
			Disk:   resources.Disk * resources.Disk,
			GPU:    resources.GPU * resources.GPU,
		})
	}

	// Add min resource stats to properties
	properties["resource_stats_min_cpu"] = minResources.CPU
	properties["resource_stats_min_memory"] = minResources.Memory
	properties["resource_stats_min_disk"] = minResources.Disk
	properties["resource_stats_min_gpu"] = minResources.GPU

	// Add max resource stats to properties
	properties["resource_stats_max_cpu"] = maxResources.CPU
	properties["resource_stats_max_memory"] = maxResources.Memory
	properties["resource_stats_max_disk"] = maxResources.Disk
	properties["resource_stats_max_gpu"] = maxResources.GPU

	// Add average and std dev resource stats to properties
	if nodesWithResources > 0 {
		// Calculate and add average resource stats
		properties["resource_stats_avg_cpu"] = sumResources.CPU / float64(nodesWithResources)
		properties["resource_stats_avg_memory"] = uint64(float64(sumResources.Memory) / float64(nodesWithResources))
		properties["resource_stats_avg_disk"] = uint64(float64(sumResources.Disk) / float64(nodesWithResources))
		properties["resource_stats_avg_gpu"] = uint64(float64(sumResources.GPU) / float64(nodesWithResources))

		// Calculate and add standard deviation resource stats
		properties["resource_stats_std_dev_cpu"] = calculateStdDev(sumSquaredResources.CPU, sumResources.CPU, nodesWithResources)
		properties["resource_stats_std_dev_memory"] =
			uint64(calculateStdDev(float64(sumSquaredResources.Memory), float64(sumResources.Memory), nodesWithResources))
		properties["resource_stats_std_dev_disk"] =
			uint64(calculateStdDev(float64(sumSquaredResources.Disk), float64(sumResources.Disk), nodesWithResources))
		properties["resource_stats_std_dev_gpu"] =
			uint64(calculateStdDev(float64(sumSquaredResources.GPU), float64(sumResources.GPU), nodesWithResources))
	} else {
		// Set zero values if no nodes with resources
		properties["resource_stats_avg_cpu"] = 0.0
		properties["resource_stats_avg_memory"] = uint64(0)
		properties["resource_stats_avg_disk"] = uint64(0)
		properties["resource_stats_avg_gpu"] = uint64(0)

		properties["resource_stats_std_dev_cpu"] = 0.0
		properties["resource_stats_std_dev_memory"] = uint64(0)
		properties["resource_stats_std_dev_disk"] = uint64(0)
		properties["resource_stats_std_dev_gpu"] = uint64(0)
	}
}

// populateCapabilitiesProperties collects capability information from nodes and adds them to properties
// Properties populated:
//   - capabilities_execution_engines_count: number of distinct execution engines
//   - capabilities_execution_engines_<engine_name>: count of nodes with this engine
//   - capabilities_storage_sources_count: number of distinct storage sources
//   - capabilities_storage_sources_<source_name>: count of nodes with this source
//   - capabilities_publishers_count: number of distinct publishers
//   - capabilities_publishers_<publisher_name>: count of nodes with this publisher
//   - capabilities_protocols_count: number of distinct protocols
//   - capabilities_protocols_<protocol_name>: count of nodes with this protocol
//   - capabilities_versions_count: number of distinct Bacalhau versions
//   - capabilities_versions_<version>: count of nodes with this version
func populateCapabilitiesProperties(nodes []models.NodeState, properties EventProperties) {
	executionEngines := make(map[string]int)
	storageSources := make(map[string]int)
	publishers := make(map[string]int)
	protocols := make(map[string]int)
	versions := make(map[string]int)

	for _, node := range nodes {
		// Track node capabilities
		for _, engine := range node.Info.ComputeNodeInfo.ExecutionEngines {
			executionEngines[engine]++
		}
		for _, source := range node.Info.ComputeNodeInfo.StorageSources {
			storageSources[source]++
		}
		for _, publisher := range node.Info.ComputeNodeInfo.Publishers {
			publishers[publisher]++
		}
		for _, protocol := range node.Info.SupportedProtocols {
			protocols[protocol.String()]++
		}
		versions[node.Info.BacalhauVersion.GitVersion]++
	}

	// Add capability counts to properties
	properties["capabilities_execution_engines_count"] = len(executionEngines)
	properties["capabilities_storage_sources_count"] = len(storageSources)
	properties["capabilities_publishers_count"] = len(publishers)
	properties["capabilities_protocols_count"] = len(protocols)
	properties["capabilities_versions_count"] = len(versions)

	// Add top capabilities for each type (limited to avoid event size issues)
	addCapabilityDetails(executionEngines, "capabilities_execution_engines_", properties)
	addCapabilityDetails(storageSources, "capabilities_storage_sources_", properties)
	addCapabilityDetails(publishers, "capabilities_publishers_", properties)
	addCapabilityDetails(protocols, "capabilities_protocols_", properties)
	addCapabilityDetails(versions, "capabilities_versions_", properties)
}

// addCapabilityDetails adds top capabilities of a specific type to properties
// For each capability in the map, adds a property with key <prefix><capability_name> = count
func addCapabilityDetails(capMap map[string]int, prefix string, properties EventProperties) {
	count := 0
	for name, num := range capMap {
		properties[prefix+name] = num
		count++
		if count >= maxEnginesToReport { // Limit to top 10 to avoid event size issues
			break
		}
	}
}

// populateUtilizationProperties computes utilization metrics for nodes and adds them to properties
// Properties populated:
//   - utilization_total_running_executions: total number of running executions
//   - utilization_total_enqueued_executions: total number of enqueued executions
//   - utilization_avg_running_executions: average running executions per node
//   - utilization_avg_enqueued_executions: average enqueued executions per node
//   - utilization_cpu_percent: percentage of total CPU resources in use
//   - utilization_memory_percent: percentage of total memory resources in use
//   - utilization_disk_percent: percentage of total disk resources in use
//   - utilization_gpu_percent: percentage of total GPU resources in use
func populateUtilizationProperties(nodes []models.NodeState, totalResources models.Resources, properties EventProperties) {
	totalRunningExecutions := 0
	totalEnqueuedExecutions := 0
	nodeCount := len(nodes)

	// Gather utilization information
	for _, node := range nodes {
		totalRunningExecutions += node.Info.ComputeNodeInfo.RunningExecutions
		totalEnqueuedExecutions += node.Info.ComputeNodeInfo.EnqueuedExecutions
	}

	// Add execution counts to properties
	properties["utilization_total_running_executions"] = totalRunningExecutions
	properties["utilization_total_enqueued_executions"] = totalEnqueuedExecutions

	// Calculate and add average executions with divide-by-zero protection
	if nodeCount > 0 {
		properties["utilization_avg_running_executions"] = float64(totalRunningExecutions) / float64(nodeCount)
		properties["utilization_avg_enqueued_executions"] = float64(totalEnqueuedExecutions) / float64(nodeCount)
	} else {
		properties["utilization_avg_running_executions"] = 0.0
		properties["utilization_avg_enqueued_executions"] = 0.0
	}

	// Calculate and add utilization percentages with divide-by-zero protection
	if totalResources.CPU > 0 {
		properties["utilization_cpu_percent"] = float64(totalRunningExecutions) / float64(totalResources.CPU) * 100
	} else {
		properties["utilization_cpu_percent"] = 0.0
	}

	if totalResources.Memory > 0 {
		properties["utilization_memory_percent"] = float64(totalRunningExecutions) / float64(totalResources.Memory) * 100
	} else {
		properties["utilization_memory_percent"] = 0.0
	}

	if totalResources.Disk > 0 {
		properties["utilization_disk_percent"] = float64(totalRunningExecutions) / float64(totalResources.Disk) * 100
	} else {
		properties["utilization_disk_percent"] = 0.0
	}

	if totalResources.GPU > 0 {
		properties["utilization_gpu_percent"] = float64(totalRunningExecutions) / float64(totalResources.GPU) * 100
	} else {
		properties["utilization_gpu_percent"] = 0.0
	}
}

// calculateStdDev calculates the standard deviation of a set of values
func calculateStdDev(sumSquared, sum float64, count int) float64 {
	if count <= 1 {
		return 0
	}
	mean := sum / float64(count)
	variance := (sumSquared/float64(count) - mean*mean) * float64(count) / float64(count-1)
	return math.Sqrt(variance)
}
