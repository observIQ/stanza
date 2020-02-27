package plugin

func newFakeNullOutput() *NullOutput {
	return &NullOutput{
		DefaultPlugin: DefaultPlugin{
			id:         "testnull",
			pluginType: "null",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
	}
}
