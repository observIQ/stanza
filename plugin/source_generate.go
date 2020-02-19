package plugin

func init() {
	registerConfig("generate", &GenerateSourceConfig{})
}

type GenerateSourceConfig struct {
	Output  string
	Message string
	Rate    float64
}
