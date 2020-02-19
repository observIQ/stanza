package plugin

func init() {
	RegisterConfig("generate", &GenerateSourceConfig{})
}

type GenerateSourceConfig struct {
	Output  string
	Message string
	Rate    float64
}
