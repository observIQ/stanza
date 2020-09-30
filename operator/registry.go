package operator

// defaultRegistry is a global registry of operator types to operator builders.
var defaultRegistry = NewRegistry()

type Registry struct {
	operators map[string]func() Builder
	plugins   map[string]func() Builder
}

func NewRegistry() *Registry {
	return &Registry{
		operators: make(map[string]func() Builder),
		plugins:   make(map[string]func() Builder),
	}
}

// RegisterOperator will register a function to an operator type.
// This function will return a builder for the supplied type.
func (r *Registry) RegisterOperator(operatorType string, newBuilder func() Builder) {
	r.operators[operatorType] = newBuilder
}

// RegisterPlugin will register a function to an plugin type.
// This function will return a builder for the supplied type.
func (r *Registry) RegisterPlugin(pluginName string, newBuilder func() Builder) {
	r.plugins[pluginName] = newBuilder
}

// Lookup looks up a given config type, prioritizing builtin operators
// before looking in registered plugins. Its second return value will
// be false if no builder is registered for that type
func (r *Registry) Lookup(configType string) (func() Builder, bool) {
	operator, ok := r.operators[configType]
	if ok {
		return operator, ok
	}

	plugin, ok := r.plugins[configType]
	if ok {
		return plugin, ok
	}

	return nil, false
}

// RegisterOperator will register an operator in the default registry
func RegisterOperator(operatorType string, newBuilder func() Builder) {
	defaultRegistry.RegisterOperator(operatorType, newBuilder)
}

// RegisterPlugin will register a plugin in the default registry
func RegisterPlugin(pluginName string, newBuilder func() Builder) {
	defaultRegistry.RegisterPlugin(pluginName, newBuilder)
}

// Lookup looks up a given config type, prioritizing builtin operators
// before looking in registered plugins. Its second return value will
// be false if no builder is registered for that type
func Lookup(configType string) (func() Builder, bool) {
	operator, ok := r.operators[configType]
	if ok {
		return operator, ok
	}

	plugin, ok := r.plugins[configType]
	if ok {
		return plugin, ok
	}

	return nil, false
}
