package ranking

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

// Constants to normalize resource values to a comparable scale.
const (
	// Scaling factors for different resource types to normalize them to a comparable scale.
	memoryScale = 1e6 // Scale memory to megabytes
	diskScale   = 1e9 // Scale disk to gigabytes
	gpuScale    = 1   // No scaling needed for GPU as it's already in units
	cpuScale    = 1   // No scaling needed for CPU as it's already in cores

	// Weights for different resource types. These values are chosen to reflect
	// the relative importance of each resource type in the context of node capacity.
	// CPU is considered the most critical resource, followed by memory, disk, and GPU.
	defaultCPUWeight    = 0.4 // CPU has the highest weight as it directly affects computation speed.
	defaultMemoryWeight = 0.3 // Memory is crucial for handling large datasets and multitasking.
	defaultDiskWeight   = 0.2 // Disk space is important but less critical than CPU and memory for most tasks.
	defaultGPUWeight    = 0.1 // GPU is given the lowest weight as not all tasks require GPU processing.

	// Rank range constants
	maxAvailableCapacityRank = 30
	maxQueueCapacityRank     = 15
)

// AvailableCapacityNodeRanker ranks nodes based on their available capacity and queue used capacity.
type AvailableCapacityNodeRanker struct{}

// NewAvailableCapacityNodeRanker creates a new instance of AvailableCapacityNodeRanker.
func NewAvailableCapacityNodeRanker() *AvailableCapacityNodeRanker {
	return &AvailableCapacityNodeRanker{}
}

// dynamicWeights calculates the weights for resources based on the job requirements.
func dynamicWeights(jobRequirements *models.Resources) (float64, float64, float64, float64) {
	// Normalize the resource values
	normalizedCPU := jobRequirements.CPU / cpuScale
	normalizedMemory := float64(jobRequirements.Memory) / memoryScale
	normalizedDisk := float64(jobRequirements.Disk) / diskScale
	normalizedGPU := float64(jobRequirements.GPU) / gpuScale

	// Calculate the total normalized resource value
	total := normalizedCPU + normalizedMemory + normalizedDisk + normalizedGPU
	if total == 0 {
		// Return default weights if job requirements are all zero
		return defaultCPUWeight, defaultMemoryWeight, defaultDiskWeight, defaultGPUWeight
	}

	// Calculate and return dynamic weights based on normalized resource values
	return normalizedCPU / total,
		normalizedMemory / total,
		normalizedDisk / total,
		normalizedGPU / total
}

// weightedCapacity calculates the weighted capacity of a node's resources using dynamic weights.
func weightedCapacity(resources models.Resources, cpuWeight, memoryWeight, diskWeight, gpuWeight float64) float64 {
	normalizedCPU := resources.CPU / cpuScale
	normalizedMemory := float64(resources.Memory) / memoryScale
	normalizedDisk := float64(resources.Disk) / diskScale
	normalizedGPU := float64(resources.GPU) / gpuScale

	return (normalizedCPU * cpuWeight) +
		(normalizedMemory * memoryWeight) +
		(normalizedDisk * diskWeight) +
		(normalizedGPU * gpuWeight)
}

// RankNodes ranks nodes based on their available capacity and queue used capacity.
// Nodes with more available capacity are ranked higher, and nodes with more queue capacity are ranked lower.
func (s *AvailableCapacityNodeRanker) RankNodes(
	ctx context.Context, job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	// Get dynamic weights based on job requirements
	jobResources, err := job.Task().ResourcesConfig.ToResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get job resources: %w", err)
	}
	cpuWeight, memoryWeight, diskWeight, gpuWeight := dynamicWeights(jobResources)

	// Initialize variables to track maximum weighted capacities
	var maxWeightedAvailableCapacity, maxQueueUsedCapacity float64
	weightedAvailableCapacities := make(map[string]float64, len(nodes))
	weightedQueueCapacities := make(map[string]float64, len(nodes))

	// Calculate weighted capacities for each node and determine the maximum values
	for _, node := range nodes {
		weightedAvailableCapacity := weightedCapacity(node.ComputeNodeInfo.AvailableCapacity, cpuWeight, memoryWeight, diskWeight, gpuWeight)
		weightedQueueUsedCapacity := weightedCapacity(node.ComputeNodeInfo.QueueUsedCapacity, cpuWeight, memoryWeight, diskWeight, gpuWeight)

		weightedAvailableCapacities[node.NodeID] = weightedAvailableCapacity
		weightedQueueCapacities[node.NodeID] = weightedQueueUsedCapacity

		if weightedAvailableCapacity > maxWeightedAvailableCapacity {
			maxWeightedAvailableCapacity = weightedAvailableCapacity
		}
		if weightedQueueUsedCapacity > maxQueueUsedCapacity {
			maxQueueUsedCapacity = weightedQueueUsedCapacity
		}
	}

	// Rank nodes based on normalized weighted capacities
	ranks := make([]orchestrator.NodeRank, len(nodes))

	for i, node := range nodes {
		weightedAvailableCapacity := weightedAvailableCapacities[node.NodeID]
		weightedQueueUsedCapacity := weightedQueueCapacities[node.NodeID]

		// Calculate the ratios of available and queue capacities
		availableRatio := 0.0
		queueRatio := 0.0

		if maxWeightedAvailableCapacity > 0 {
			availableRatio = weightedAvailableCapacity / maxWeightedAvailableCapacity
		}
		if maxQueueUsedCapacity > 0 {
			queueRatio = weightedQueueUsedCapacity / maxQueueUsedCapacity
		}

		// Normalize the ratios to the rank range
		normalizedAvailableRank := availableRatio * float64(maxAvailableCapacityRank)
		normalizedQueueRank := (1 - queueRatio) * float64(maxQueueCapacityRank)

		// Calculate the final rank, higher available capacity and lower queue used capacity should give a higher rank
		rank := normalizedAvailableRank + normalizedQueueRank

		// Ensure the rank is within the desired range
		rank = math.Max(rank, float64(orchestrator.RankPossible))
		rank = math.Min(rank, maxAvailableCapacityRank+maxQueueCapacityRank)

		// Assign rank and reason to the node
		ranks[i] = orchestrator.NodeRank{
			NodeInfo: node,
			Rank:     int(rank),
			Reason: fmt.Sprintf(
				"Ranked based on available capacity %s and queue capacity %s",
				node.ComputeNodeInfo.AvailableCapacity.String(), node.ComputeNodeInfo.QueueUsedCapacity.String()),
			Retryable: true,
		}
		log.Ctx(ctx).Trace().Object("Rank", ranks[i]).Msg("Ranked node")
	}

	return ranks, nil
}
