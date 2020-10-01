package operator

// DefaultRegistry is a global registry of operator types to operator builders.
var DefaultRegistry = NewRegistry()

type Registry struct {
	operators map[string]func() Builder
	plugins   map[string]func() MultiBuilder
}

func NewRegistry() *Registry {
	return &Registry{
		operators: make(map[string]func() Builder),
		plugins:   make(map[string]func() MultiBuilder),
	}
}

// RegisterOperator will register a function to an operator type.
// This function will return a builder for the supplied type.
func (r *Registry) RegisterOperator(operatorType string, newBuilder func() Builder) {
	r.operators[operatorType] = newBuilder
}

// RegisterPlugin will register a function to an plugin type.
// This function will return a builder for the supplied type.
func (r *Registry) RegisterPlugin(pluginName string, newBuilder func() MultiBuilder) {
	r.plugins[pluginName] = newBuilder
}

// Lookup looks up a given config type, prioritizing builtin operators
// before looking in registered plugins. Its second return value will
// be false if no builder is registered for that type
func (r *Registry) Lookup(configType string) (func() MultiBuilder, bool) {
	b, ok := r.operators[configType]
	if ok {
		return WrapBuilder(b), ok
	}

	mb, ok := r.plugins[configType]
	if ok {
		return mb, ok
	}

	return nil, false
}

// RegisterOperator will register an operator in the default registry
func RegisterOperator(operatorType string, newBuilder func() Builder) {
	DefaultRegistry.RegisterOperator(operatorType, newBuilder)
}

// RegisterPlugin will register a plugin in the default registry
func RegisterPlugin(pluginName string, newMultiBuilder func() MultiBuilder) {
	DefaultRegistry.RegisterPlugin(pluginName, newMultiBuilder)
}

// Lookup looks up a given config type, prioritizing builtin operators
// before looking in registered plugins. Its second return value will
// be false if no builder is registered for that type
func Lookup(configType string) (func() MultiBuilder, bool) {
	return DefaultRegistry.Lookup(configType)
}

func WrapBuilder(f func() Builder) func() MultiBuilder {
	return func() MultiBuilder {
		return &MultiBuilderWrapper{f()}
	}
}

type MultiBuilderWrapper struct {
	Builder
}

func (m *MultiBuilderWrapper) BuildMulti(bc BuildContext) ([]Operator, error) {
	op, err := m.Builder.Build(bc)
	return []Operator{op}, err
}

func (m *MultiBuilderWrapper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(m.Builder)
}
