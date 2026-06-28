package project

const (
	DefaultSourceDir = "src"
	DefaultOutputDir = "dist"
)

type Config struct {
	Pack      PackConfig      `toml:"pack"`
	Minecraft MinecraftConfig `toml:"minecraft"`
	Build     BuildConfig     `toml:"build"`
}

type PackConfig struct {
	ID   string `toml:"id"`
	Name string `toml:"name"`
}

type MinecraftConfig struct {
	Versions   string `toml:"versions"`
	Target     string `toml:"target"`
	PackFormat int    `toml:"pack_format"`
}

type BuildConfig struct {
	Source string `toml:"source"`
	Output string `toml:"output"`
}

func (config *Config) ApplyDefaults() {
	if config.Pack.Name == "" {
		config.Pack.Name = config.Pack.ID
	}

	if config.Build.Source == "" {
		config.Build.Source = DefaultSourceDir
	}
	if config.Build.Output == "" {
		config.Build.Output = DefaultOutputDir
	}
}
