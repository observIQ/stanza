# Contributing to bplogagent

## Development

You can view and edit the source code by cloning this repository:

```bash
git clone https://github.com/observiq/bplogagent.git
```
## Pull Requests

### How to Send Pull Requests

Everyone is welcome to contribute code to `bplogagent` via
GitHub pull requests (PRs).

To create a new PR, fork the project in GitHub and clone the upstream
repo:

```sh
$ go get -d github.com/observiq/bplogagent
```

This will put the project in `${GOPATH}/src/github.com/observiq/bplogagent`. You
can alternatively use `git` directly with:

```sh
$ git clone https://github.com/observiq/bplogagent
```

This would put the project in the `bplogagent` directory in
current working directory.

Enter the newly created directory and add your fork as a new remote:

```sh
$ git remote add <YOUR_FORK> git@github.com:<YOUR_GITHUB_USERNAME>/opentelemetry-go
```

Check out a new branch, make modifications, run linters and tests, and
push the branch to your fork:

```sh
$ git checkout -b <YOUR_BRANCH_NAME>
# edit files
$ go test -race ./...
$ git add -p
$ git commit
$ git push <YOUR_FORK> <YOUR_BRANCH_NAME>
```

Open a pull request against the main `bplogagent` repo.

### How to Receive Comments

* If the PR is not ready for review, please put `[WIP]` in the title,
  tag it as `work-in-progress`, or mark it as
  [`draft`](https://github.blog/2019-02-14-introducing-draft-pull-requests/).
* Make sure CI passes.

### How to Get PRs Merged

A PR is considered to be **ready to merge** when:

* It has received approval from at least one maintainer.
* CI passes.
* Major feedback is resolved.

## Design Choices

Best practices for plugin development are documented below, but for changes to
the core agent, we are happy to discuss proposals in the issue tracker.

### Plugin Development

In order to build a plugin, follow these three steps:
1. Build a unique plugin struct that satisfies the [`Plugin`](plugin/plugin.go) interface. This struct will define what your plugin does when executed in the pipeline.

```go
type ExamplePlugin struct {
	FilePath string
}

func (p *ExamplePlugin) Process(ctx context.Context, entry *entry.Entry) error {
	// Processing logic
}
```

2. Build a unique config struct that satisfies the [`Config`](plugin/config.go) interface. This struct will define the parameters used to configure and build your plugin struct in step 1.

```go
type ExamplePluginConfig struct {
	filePath string
}

func (c ExamplePluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	return &ExamplePlugin{
		filePath: c.FilePath,
	}, nil
}
```

3. Register your config struct in the plugin registry using an `init()` hook. This will ensure that the agent knows about your plugin at runtime and can build it from a YAML config.

```go
func init() {
	plugin.Register("example_plugin", &ExamplePluginConfig{})
}
```

## Any tips for building plugins?
We highly recommend that developers take advantage of [helpers](plugin/helper) when building their plugins. Helpers are structs that help satisfy common behavior shared across many plugins. By embedding these structs, you can skip having to satisfy certain aspects of the `plugin` and `config` interfaces.

For example, almost all plugins should embed the [BasicPlugin](plugin/helper/basic_plugin.go) helper, as it provides simple functionality for returning a plugin id and plugin type.

```go
// ExamplePlugin is a basic plugin, with a basic lifecycle, that consumes
// but doesn't send log entries. Rather than implementing every part of the plugin
// interface, we can embed the following helpers to achieve this effect.
type ExamplePlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicOutput
}
```
