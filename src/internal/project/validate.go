package project

import (
	"errors"
	"fmt"
	"regexp"
)

var packIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*`)

func ValidateConfig(config *Config) error {
	if config == nil {
		return errors.New("config is nil")
	}

	if config.Pack.ID == "" {
		return errors.New("pack.id is required")
	}

	if !packIDPattern.MatchString(config.Pack.ID) {
		return fmt.Errorf("pack.id must be snake_case, got %q", config.Pack.ID)
	}

	if config.Minecraft.Versions == "" {
		return errors.New("minecraft.versions is required")
	}

	if config.Build.Source == "" {
		return errors.New("build.source is required")
	}

	if config.Build.Output == "" {
		return errors.New("build.output is required")
	}

	if config.Minecraft.PackFormat < 0 {
		return errors.New("minecraft.pack_format cannot be negative")
	}

	return nil
}
