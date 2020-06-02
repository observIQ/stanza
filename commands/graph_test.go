package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func ExampleGraph() {
	config := []byte(`
plugins:
  - id: generate
    type: generate_input
    output: json_parser
    record:
      test: value

  - id: json_parser
    type: json_parser
    output: google_cloud

  - id: google_cloud
    project_id: testproject
    type: google_cloud_output
`)
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")
	err = ioutil.WriteFile(configPath, config, 0666)
	if err != nil {
		panic(err)
	}

	rootFlags := &RootFlags{
		ConfigFiles: []string{configPath},
	}
	graphCmd := NewGraphCommand(rootFlags)
	graphCmd.Execute()

	// Output:
	// strict digraph G {
	//  // Node definitions.
	//  generate;
	//  json_parser;
	//  google_cloud;
	//
	//  // Edge definitions.
	//  generate -> json_parser;
	//  json_parser -> google_cloud;
	// }

}
