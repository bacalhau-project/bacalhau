//go:build unit || !integration

package translation_test

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
	"github.com/stretchr/testify/suite"
)

type TranslationTestSuite struct {
	suite.Suite
	ctx      context.Context
	provider translation.TranslatorProvider
}

func TestTranslationTestSuite(t *testing.T) {
	suite.Run(t, new(TranslationTestSuite))
}

func (s *TranslationTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.provider = translation.NewStandardTranslators()
}

var testcases = []struct {
	name     string
	spec     *models.SpecConfig
	expected *models.SpecConfig
}{
	{
		name: "python",
		spec: &models.SpecConfig{
			Type: "python",
			Params: map[string]interface{}{
				"Command":   "python",
				"Arguments": []interface{}{"-c", "print('Hello, world!')"},
			},
		},
		expected: &models.SpecConfig{
			Type: "docker",
			Params: map[string]interface{}{
				"Image":      "rossjones73/exec-python-3.11:0.5",
				"Entrypoint": []string{},
				"Parameters": []string{
					"/build/launcher.py", "--", "python", "-c", "print('Hello, world!')",
				},
				"EnvironmentVariables": []string{},
				"WorkingDirectory":     "",
			},
		},
	},
}

func (s *TranslationTestSuite) TestTranslate() {
	for _, tc := range testcases {
		s.Run(tc.name, func() {
			job := &models.Job{
				ID: tc.name,
				Tasks: []*models.Task{
					{
						Name:   "task1",
						Engine: tc.spec,
					},
				},
			}

			translated, err := translation.Translate(s.ctx, s.provider, job)
			s.Require().NoError(err)

			s.Require().Equal(tc.expected, translated.Task().Engine)
		})
	}
}

func (s *TranslationTestSuite) TestTranslateWithInvalidEngine() {
	job := &models.Job{
		ID: "invalid_engine",
		Tasks: []*models.Task{
			{
				Name: "task1",
				Engine: &models.SpecConfig{
					Type: "invalid",
				},
			},
		},
	}

	_, err := translation.Translate(s.ctx, s.provider, job)
	s.Require().Error(err)
}

func (s *TranslationTestSuite) TestTranslateWithDefaultEngine() {
	job := &models.Job{
		ID: "invalid_engine",
		Tasks: []*models.Task{
			{
				Name: "task1",
				Engine: &models.SpecConfig{
					Type: "docker",
				},
			},
		},
	}

	translated, err := translation.Translate(s.ctx, s.provider, job)
	s.Require().NoError(err)
	s.Require().Nil(translated)
}

func (s *TranslationTestSuite) TestTranslateWithMixedEngines() {
	job := &models.Job{
		ID: "invalid_engine",
		Tasks: []*models.Task{
			{
				Name: "task1",
				Engine: &models.SpecConfig{
					Type: "docker",
				},
			},
			{
				Name: "task2",
				Engine: &models.SpecConfig{
					Type: "duckdb",
					Params: map[string]interface{}{
						"Command":   "duckdb",
						"Arguments": []interface{}{"-csv", "-c", "select * from table;"},
					},
				},
			},
		},
	}

	translated, err := translation.Translate(s.ctx, s.provider, job)
	s.Require().NoError(err)
	s.Require().NotNil(translated)

	// Before
	s.Require().Equal("docker", job.Tasks[0].Engine.Type)
	s.Require().Equal("duckdb", job.Tasks[1].Engine.Type)

	// After
	s.Require().Equal("docker", translated.Tasks[0].Engine.Type)
	s.Require().Equal("docker", translated.Tasks[1].Engine.Type)
}

func (s *TranslationTestSuite) TestShouldTranslateWithDefaultEngine() {
	tasks := []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: "docker",
			},
		},
	}

	should, err := translation.ShouldTranslate(s.ctx, s.provider, tasks)
	s.Require().NoError(err)
	s.Require().False(should)
}

func (s *TranslationTestSuite) TestShouldTranslateWithNonDefaultEngine() {
	tasks := []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: "python",
			},
		},
	}

	should, err := translation.ShouldTranslate(s.ctx, s.provider, tasks)
	s.Require().NoError(err)
	s.Require().True(should)
}
