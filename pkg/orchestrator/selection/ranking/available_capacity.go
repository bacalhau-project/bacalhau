package ranking

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
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
	maxAvailableCapacityRank = 80
	maxQueueCapacityRank     = 20
)

// resourceWeights struct to hold resource weights.
type resourceWeights struct {
	cpuWeight    float64
	memoryWeight float64
	diskWeight   float64
	gpuWeight    float64
}

// defaultResourceWeights returns the default resource weights.
var defaultResourceWeights = resourceWeights{
	cpuWeight:    defaultCPUWeight,
	memoryWeight: defaultMemoryWeight,
	diskWeight:   defaultDiskWeight,
	gpuWeight:    defaultGPUWeight,
}

// AvailableCapacityNodeRanker ranks nodes based on their available capacity and queue used capacity.
type AvailableCapacityNodeRanker struct{}

// NewAvailableCapacityNodeRanker creates a new instance of AvailableCapacityNodeRanker.
func NewAvailableCapacityNodeRanker() *AvailableCapacityNodeRanker {
	return &AvailableCapacityNodeRanker{}
}

// dynamicWeights calculates the weights for resources based on the job requirements.
func dynamicWeights(jobRequirements *models.Resources) resourceWeights {
	// Normalize the resource values
	normalizedCPU := jobRequirements.CPU / cpuScale
	normalizedMemory := float64(jobRequirements.Memory) / memoryScale
	normalizedDisk := float64(jobRequirements.Disk) / diskScale
	normalizedGPU := float64(jobRequirements.GPU) / gpuScale

	// Calculate the total normalized resource value
	total := normalizedCPU + normalizedMemory + normalizedDisk + normalizedGPU
	if total == 0 {
		// Return default weights if job requirements are all zero
		return defaultResourceWeights
	}

	// Calculate and return dynamic weights based on normalized resource values
	return resourceWeights{
		cpuWeight:    normalizedCPU / total,
		memoryWeight: normalizedMemory / total,
		diskWeight:   normalizedDisk / total,
		gpuWeight:    normalizedGPU / total,
	}
}

func weightedCapacity(resources models.Resources, weights resourceWeights) float64 {
	normalizedCPU := resources.CPU / cpuScale
	normalizedMemory := float64(resources.Memory) / memoryScale
	normalizedDisk := float64(resources.Disk) / diskScale
	normalizedGPU := float64(resources.GPU) / gpuScale

	return (normalizedCPU * weights.cpuWeight) +
		(normalizedMemory * weights.memoryWeight) +
		(normalizedDisk * weights.diskWeight) +
		(normalizedGPU * weights.gpuWeight)
}

// calculateWeightedCapacities calculates the weighted capacities for each node and determines the maximum values
func (s *AvailableCapacityNodeRanker) calculateWeightedCapacities(nodes []models.NodeInfo, weights resourceWeights) (
	map[string]float64, map[string]float64, float64, float64) {
	var maxWeightedAvailableCapacity, maxQueueUsedCapacity float64
	weightedAvailableCapacities := make(map[string]float64, len(nodes))
	weightedQueueCapacities := make(map[string]float64, len(nodes))

	for _, node := range nodes {
		weightedAvailableCapacity := weightedCapacity(node.ComputeNodeInfo.AvailableCapacity, weights)
		weightedQueueUsedCapacity := weightedCapacity(node.ComputeNodeInfo.QueueUsedCapacity, weights)

		weightedAvailableCapacities[node.ID()] = weightedAvailableCapacity
		weightedQueueCapacities[node.ID()] = weightedQueueUsedCapacity

		if weightedAvailableCapacity > maxWeightedAvailableCapacity {
			maxWeightedAvailableCapacity = weightedAvailableCapacity
		}
		if weightedQueueUsedCapacity > maxQueueUsedCapacity {
			maxQueueUsedCapacity = weightedQueueUsedCapacity
		}
	}

	return weightedAvailableCapacities, weightedQueueCapacities, maxWeightedAvailableCapacity, maxQueueUsedCapacity
}

// rankNodesBasedOnCapacities ranks nodes based on normalized weighted capacities
func (s *AvailableCapacityNodeRanker) rankNodesBasedOnCapacities(ctx context.Context, nodes []models.NodeInfo,
	wAvailableCapacities, wQueueCapacities map[string]float64, maxAvailableCapacity, maxQueueCapacity float64) (
	[]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))

	for i, node := range nodes {
		weightedAvailableCapacity := wAvailableCapacities[node.ID()]
		weightedQueueUsedCapacity := wQueueCapacities[node.ID()]

		// Calculate the ratios of available and queue capacities
		availableRatio := 0.0
		queueRatio := 0.0

		if maxAvailableCapacity > 0 {
			availableRatio = weightedAvailableCapacity / maxAvailableCapacity
		}
		if maxQueueCapacity > 0 {
			queueRatio = weightedQueueUsedCapacity / maxQueueCapacity
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

// RankNodes ranks nodes based on their available capacity and queue used capacity.
// Nodes with more available capacity are ranked higher, and nodes with more queue capacity are ranked lower.
func (s *AvailableCapacityNodeRanker) RankNodes(
	ctx context.Context, job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	// Get dynamic weights based on job requirements
	jobResources, err := job.Task().ResourcesConfig.ToResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get job resources: %w", err)
	}
	weights := dynamicWeights(jobResources)

	// Calculate weighted capacities for each node and determine the maximum values
	wAvailableCapacities, wQueueCapacities, maxAvailableCapacity, maxQueueCapacity :=
		s.calculateWeightedCapacities(nodes, weights)

	// Rank nodes based on normalized weighted capacities
	return s.rankNodesBasedOnCapacities(
		ctx, nodes, wAvailableCapacities, wQueueCapacities, maxAvailableCapacity, maxQueueCapacity)
}
