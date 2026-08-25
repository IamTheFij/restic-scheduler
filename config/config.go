package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"git.iamthefij.com/iamthefij/restic-scheduler/tasks"
)

var ErrNoJobsFound = errors.New("no jobs found and at least one job is required")

// Config is the global configuration for the scheduler containing job configuration.
type Config struct {
	DefaultConfig *tasks.ResticConfig `hcl:"default_config,block"`
	Jobs          []tasks.Job         `hcl:"job,block"`
}

// Validate ensures that the scheduler configuration is valid
func (c Config) Validate() error {
	if len(c.Jobs) == 0 {
		return ErrNoJobsFound
	}

	for _, job := range c.Jobs {
		// Use default restic config if no job config is provided
		// TODO: Maybe merge values here
		if job.Config == nil {
			job.Config = c.DefaultConfig
		}

		if err := job.Validate(); err != nil {
			return fmt.Errorf("failed config validation: %w", err)
		}
	}

	return nil
}

func ParseConfig(path string, src []byte) ([]tasks.Job, error) {
	var config Config

	ctx := hcl.EvalContext{
		Variables: nil,
		Functions: map[string]function.Function{
			"env": function.New(&function.Spec{
				Params: []function.Parameter{{
					Name:             "var",
					Type:             cty.String,
					AllowNull:        false,
					AllowUnknown:     false,
					AllowDynamicType: false,
					AllowMarked:      false,
				}},
				VarParam: nil,
				Type:     function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					return cty.StringVal(os.Getenv(args[0].AsString())), nil
				},
			}),
			"readfile": function.New(&function.Spec{
				Params: []function.Parameter{{
					Name:             "path",
					Type:             cty.String,
					AllowNull:        false,
					AllowUnknown:     false,
					AllowDynamicType: false,
					AllowMarked:      false,
				}},
				VarParam: nil,
				Type:     function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					content, err := os.ReadFile(args[0].AsString())
					if err != nil {
						return cty.StringVal(""), err
					}

					return cty.StringVal(string(content)), nil
				},
			}),
		},
	}

	if err := hclsimple.Decode(path, src, &ctx, &config); err != nil {
		return nil, fmt.Errorf("failed to decode file %s: %w", path, err)
	}

	if len(config.Jobs) == 0 {
		log.Printf("no jobs defined in file %s", path)

		return []tasks.Job{}, nil
	}

	for _, job := range config.Jobs {
		if err := job.Validate(); err != nil {
			return nil, fmt.Errorf("%s: Invalid job: %w", path, err)
		}
	}

	return config.Jobs, nil
}

func ParseConfigFile(path string) ([]tasks.Job, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Config file not found at %s: %w", path, err)
		}

		return nil, fmt.Errorf("couldn't read %s: %w", path, err)
	}

	return ParseConfig(path, src)
}
