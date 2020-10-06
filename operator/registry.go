package operator

import "encoding/json"

// DefaultRegistry is a global registry of operator types to operator builders.
var DefaultRegistry = NewRegistry()

// Registry is a registry for operators and plugins that is used for
// building types from IDs
type Registry struct {
	operators map[string]func() Builder
	plugins   map[string]func() MultiBuilder
}

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{
		operators: make(map[string]func() Builder),
		plugins:   make(map[string]func() MultiBuilder),
	}
}

// Register will register a function to an operator type.
// This function will return a builder for the supplied type.
func (r *Registry) Register(operatorType string, newBuilder func() Builder) {
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

// Register will register an operator in the default registry
func Register(operatorType string, newBuilder func() Builder) {
	DefaultRegistry.Register(operatorType, newBuilder)
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

// WrapBuilder takes a function that would create a Builder, and
// returns a function that makes a MultiBuilder instead
func WrapBuilder(f func() Builder) func() MultiBuilder {
	return func() MultiBuilder {
		return &MultiBuilderWrapper{f()}
	}
}

// MultiBuilderWrapper wraps a Builder to turn it into a MultiBuilder
type MultiBuilderWrapper struct {
	Builder
}

// BuildMulti implements MultiBuilder.BuildMulti
func (m *MultiBuilderWrapper) BuildMulti(bc BuildContext) ([]Operator, error) {
	op, err := m.Builder.Build(bc)
	return []Operator{op}, err
}

// UnmarshalYAML unmarshals YAML
func (m *MultiBuilderWrapper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(m.Builder)
}

// UnmarshalJSON unmarshals JSON
func (m *MultiBuilderWrapper) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, m.Builder)
}

// MarshalJSON marshalls JSON
func (m MultiBuilderWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Builder)
}

// MarshalYAML marshalls YAML
func (m MultiBuilderWrapper) MarshalYAML() (interface{}, error) {
	return m.Builder, nil
}
