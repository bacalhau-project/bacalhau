package transformer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/suite"

	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	storage_url "github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

type LegacyWasmModuleTransformerSuite struct {
	suite.Suite
	transformer *LegacyWasmModuleTransformer
	ctx         context.Context
}

func TestLegacyWasmModuleTransformerSuite(t *testing.T) {
	suite.Run(t, new(LegacyWasmModuleTransformerSuite))
}

func (s *LegacyWasmModuleTransformerSuite) SetupTest() {
	s.transformer = NewLegacyWasmModuleTransformer()
	s.ctx = context.Background()
}

func (s *LegacyWasmModuleTransformerSuite) TestIgnoresNonWasmJobs() {
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineDocker,
					Params: map[string]interface{}{},
				},
			},
		},
	}

	err := s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)
	s.Require().Equal(models.EngineDocker, job.Task().Engine.Type)
}

func (s *LegacyWasmModuleTransformerSuite) TestIgnoresNewFormatJobs() {
	// Create a job with new format
	params := map[string]interface{}{
		"EntryModule": "/entry.wasm",
		"Entrypoint":  "run",
		"Parameters":  []string{"--arg1", "value1"},
	}
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: params,
				},
			},
		},
	}

	err := s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)

	// Verify nothing changed
	s.Require().Equal("/entry.wasm", job.Task().Engine.Params["EntryModule"])
	s.Require().Equal("run", job.Task().Engine.Params["Entrypoint"])
	s.Require().Equal([]string{"--arg1", "value1"}, job.Task().Engine.Params["Parameters"])
}

func (s *LegacyWasmModuleTransformerSuite) TestTransformsLegacyFormat() {
	// Create a job with legacy format
	entryModuleSpec, err := storage_url.NewSpecConfig("https://example.com/entry.wasm")
	s.Require().NoError(err)
	entryModule := &models.InputSource{
		Source: entryModuleSpec,
		Target: "/entry.wasm",
	}

	importModuleSpec, err := storage_url.NewSpecConfig("https://example.com/import.wasm")
	s.Require().NoError(err)
	importModule := &models.InputSource{
		Source: importModuleSpec,
		Target: "/import.wasm",
	}

	params := map[string]interface{}{
		"EntryModule":   entryModule,
		"EntryPoint":    "run",
		"Parameters":    []string{"--arg1", "value1"},
		"ImportModules": []*models.InputSource{importModule},
	}
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: params,
				},
			},
		},
	}

	err = s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)

	// Verify the transformation
	spec, err := wasmmodels.DecodeSpec(job.Task().Engine)
	s.Require().NoError(err)

	// Check engine params
	s.Equal("run", spec.Entrypoint)
	s.Equal([]string{"--arg1", "value1"}, spec.Parameters)
	s.Equal("/entry.wasm", spec.EntryModule)
	s.Equal([]string{"/import.wasm"}, spec.ImportModules)

	// Check input sources
	s.Require().Len(job.Task().InputSources, 2)

	// Find entry module in input sources
	var foundEntry, foundImport bool
	for _, input := range job.Task().InputSources {
		if input.Target == "/entry.wasm" {
			s.Equal(entryModule.Source, input.Source)
			foundEntry = true
		}
		if input.Target == "/import.wasm" {
			s.Equal(importModule.Source, input.Source)
			foundImport = true
		}
	}
	s.True(foundEntry, "Entry module not found in input sources")
	s.True(foundImport, "Import module not found in input sources")
}

func (s *LegacyWasmModuleTransformerSuite) TestHandlesEmptyImportModules() {
	// Create a job with only entry module
	entryModuleSpec, err := storage_url.NewSpecConfig("https://example.com/entry.wasm")
	s.Require().NoError(err)
	entryModule := &models.InputSource{
		Source: entryModuleSpec,
		Target: "/entry.wasm",
	}

	params := map[string]interface{}{
		"EntryModule": entryModule,
		"EntryPoint":  "run",
	}
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: params,
				},
			},
		},
	}

	err = s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)

	// Verify the transformation
	spec, err := wasmmodels.DecodeSpec(job.Task().Engine)
	s.Require().NoError(err)
	s.Equal("/entry.wasm", spec.EntryModule)
	s.Empty(spec.ImportModules)
	s.Require().Len(job.Task().InputSources, 1)

	// Check input source
	input := job.Task().InputSources[0]
	s.Equal(entryModule.Source, input.Source)
	s.Equal("/entry.wasm", input.Target)
}

func (s *LegacyWasmModuleTransformerSuite) TestHandlesEmptyJob() {
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: map[string]interface{}{},
				},
			},
		},
	}
	err := s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)
	s.Empty(job.Task().InputSources)
}

func (s *LegacyWasmModuleTransformerSuite) TestValidatesInputSources() {
	// Create a job with legacy format but missing target
	entryModuleSpec, err := storage_url.NewSpecConfig("https://example.com/entry.wasm")
	s.Require().NoError(err)
	entryModule := &models.InputSource{
		Source: entryModuleSpec,
		// Missing Target
	}

	params := map[string]interface{}{
		"EntryModule": entryModule,
		"EntryPoint":  "run",
	}
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: params,
				},
			},
		},
	}

	err = s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)

	// Verify the default target was set
	spec, err := wasmmodels.DecodeSpec(job.Task().Engine)
	s.Require().NoError(err)
	s.Equal("/wasm/entry.wasm", spec.EntryModule)

	// Verify input source was added with default target
	s.Require().Len(job.Task().InputSources, 1)
	input := job.Task().InputSources[0]
	s.Equal("/wasm/entry.wasm", input.Target)
	s.Equal(entryModule.Source, input.Source)
}

func (s *LegacyWasmModuleTransformerSuite) TestValidatesImportModules() {
	// Create a job with legacy format but missing target in import module
	entryModuleSpec, err := storage_url.NewSpecConfig("https://example.com/entry.wasm")
	s.Require().NoError(err)
	entryModule := &models.InputSource{
		Source: entryModuleSpec,
		Target: "/entry.wasm",
	}

	importModuleSpec, err := storage_url.NewSpecConfig("https://example.com/import.wasm")
	s.Require().NoError(err)
	importModule := &models.InputSource{
		Source: importModuleSpec,
		// Missing Target
	}

	params := map[string]interface{}{
		"EntryModule":   entryModule,
		"EntryPoint":    "run",
		"ImportModules": []*models.InputSource{importModule},
	}
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: params,
				},
			},
		},
	}

	err = s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)

	// Verify the transformation
	spec, err := wasmmodels.DecodeSpec(job.Task().Engine)
	s.Require().NoError(err)
	s.Equal("/entry.wasm", spec.EntryModule)
	s.Equal([]string{"/wasm/imports/module_0.wasm"}, spec.ImportModules)

	// Verify input sources
	s.Require().Len(job.Task().InputSources, 2)

	// Find import module in input sources
	var foundImport bool
	for _, input := range job.Task().InputSources {
		if input.Target == "/wasm/imports/module_0.wasm" {
			s.Equal(importModule.Source, input.Source)
			foundImport = true
		}
	}
	s.True(foundImport, "Import module not found in input sources")
}

func (s *LegacyWasmModuleTransformerSuite) TestHandlesDuplicateInputSources() {
	// Create a job with an existing input source
	existingSpec, err := storage_url.NewSpecConfig("https://example.com/existing.wasm")
	s.Require().NoError(err)
	existingInput := &models.InputSource{
		Source: existingSpec,
		Target: "/entry.wasm",
	}
	job := &models.Job{
		Tasks: []*models.Task{
			{
				Engine: &models.SpecConfig{
					Type:   models.EngineWasm,
					Params: map[string]interface{}{},
				},
				InputSources: []*models.InputSource{existingInput},
			},
		},
	}

	// Add a legacy module with the same target
	entryModuleSpec, err := storage_url.NewSpecConfig("https://example.com/entry.wasm")
	s.Require().NoError(err)
	entryModule := &models.InputSource{
		Source: entryModuleSpec,
		Target: "/entry.wasm",
	}
	job.Task().Engine.Params = map[string]interface{}{
		"EntryModule": entryModule,
	}

	err = s.transformer.Transform(s.ctx, job)
	s.Require().NoError(err)

	// Verify both input sources were added
	s.Require().Len(job.Task().InputSources, 2) // Both should be added since they have different sources
	spec, err := wasmmodels.DecodeSpec(job.Task().Engine)
	s.Require().NoError(err)
	s.Require().Equal("/entry.wasm", spec.EntryModule)
}

func TestLegacyWasmModuleTransformer(t *testing.T) {
	tests := []struct {
		name     string
		job      *models.Job
		expected *models.Job
	}{
		{
			name: "not a wasm job",
			job: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineDocker,
						},
					},
				},
			},
			expected: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineDocker,
						},
					},
				},
			},
		},
		{
			name: "already in new format",
			job: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineWasm,
							Params: wasmmodels.EngineSpec{
								EntryModule: "/wasm/entry.wasm",
							}.ToMap(),
						},
					},
				},
			},
			expected: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineWasm,
							Params: wasmmodels.EngineSpec{
								EntryModule: "/wasm/entry.wasm",
							}.ToMap(),
						},
					},
				},
			},
		},
		{
			name: "legacy format with empty targets",
			job: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineWasm,
							Params: map[string]interface{}{
								"EntryModule": &models.InputSource{
									Source: &models.SpecConfig{
										Type: models.StorageSourceURL,
										Params: map[string]interface{}{
											"URL": "https://example.com/entry.wasm",
										},
									},
								},
								"ImportModules": []*models.InputSource{
									{
										Source: &models.SpecConfig{
											Type: models.StorageSourceURL,
											Params: map[string]interface{}{
												"URL": "https://example.com/import1.wasm",
											},
										},
									},
									{
										Source: &models.SpecConfig{
											Type: models.StorageSourceURL,
											Params: map[string]interface{}{
												"URL": "https://example.com/import2.wasm",
											},
										},
									},
								},
								"EntryPoint": "main",
								"Parameters": []string{"arg1", "arg2"},
							},
						},
					},
				},
			},
			expected: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineWasm,
							Params: wasmmodels.EngineSpec{
								EntryModule: "/wasm/entry.wasm",
								ImportModules: []string{
									"/wasm/imports/module_0.wasm",
									"/wasm/imports/module_1.wasm",
								},
								Entrypoint: "main",
								Parameters: []string{"arg1", "arg2"},
							}.ToMap(),
						},
						InputSources: []*models.InputSource{
							{
								Source: &models.SpecConfig{
									Type: models.StorageSourceURL,
									Params: map[string]interface{}{
										"URL": "https://example.com/entry.wasm",
									},
								},
								Target: "/wasm/entry.wasm",
							},
							{
								Source: &models.SpecConfig{
									Type: models.StorageSourceURL,
									Params: map[string]interface{}{
										"URL": "https://example.com/import1.wasm",
									},
								},
								Target: "/wasm/imports/module_0.wasm",
							},
							{
								Source: &models.SpecConfig{
									Type: models.StorageSourceURL,
									Params: map[string]interface{}{
										"URL": "https://example.com/import2.wasm",
									},
								},
								Target: "/wasm/imports/module_1.wasm",
							},
						},
					},
				},
			},
		},
		{
			name: "legacy format with existing targets",
			job: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineWasm,
							Params: map[string]interface{}{
								"EntryModule": &models.InputSource{
									Source: &models.SpecConfig{
										Type: models.StorageSourceURL,
										Params: map[string]interface{}{
											"URL": "https://example.com/entry.wasm",
										},
									},
									Target: "/custom/entry.wasm",
								},
								"ImportModules": []*models.InputSource{
									{
										Source: &models.SpecConfig{
											Type: models.StorageSourceURL,
											Params: map[string]interface{}{
												"URL": "https://example.com/import1.wasm",
											},
										},
										Target: "/custom/import1.wasm",
									},
									{
										Source: &models.SpecConfig{
											Type: models.StorageSourceURL,
											Params: map[string]interface{}{
												"URL": "https://example.com/import2.wasm",
											},
										},
										Target: "/custom/import2.wasm",
									},
								},
								"EntryPoint": "main",
								"Parameters": []string{"arg1", "arg2"},
							},
						},
					},
				},
			},
			expected: &models.Job{
				Tasks: []*models.Task{
					{
						Engine: &models.SpecConfig{
							Type: models.EngineWasm,
							Params: wasmmodels.EngineSpec{
								EntryModule: "/custom/entry.wasm",
								ImportModules: []string{
									"/custom/import1.wasm",
									"/custom/import2.wasm",
								},
								Entrypoint: "main",
								Parameters: []string{"arg1", "arg2"},
							}.ToMap(),
						},
						InputSources: []*models.InputSource{
							{
								Source: &models.SpecConfig{
									Type: models.StorageSourceURL,
									Params: map[string]interface{}{
										"URL": "https://example.com/entry.wasm",
									},
								},
								Target: "/custom/entry.wasm",
							},
							{
								Source: &models.SpecConfig{
									Type: models.StorageSourceURL,
									Params: map[string]interface{}{
										"URL": "https://example.com/import1.wasm",
									},
								},
								Target: "/custom/import1.wasm",
							},
							{
								Source: &models.SpecConfig{
									Type: models.StorageSourceURL,
									Params: map[string]interface{}{
										"URL": "https://example.com/import2.wasm",
									},
								},
								Target: "/custom/import2.wasm",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			transformer := NewLegacyWasmModuleTransformer()
			err := transformer.Transform(context.Background(), tc.job)
			require.NoError(t, err)
			require.Equal(t, tc.expected, tc.job)
		})
	}
}
