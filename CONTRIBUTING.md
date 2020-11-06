# Contributing to Stanza

## Development

You can view and edit the source code by cloning this repository:

```bash
git clone https://github.com/observiq/stanza.git
```

## Pull Requests

### How to Submit Pull Requests

Everyone is welcome to contribute code to `stanza` via GitHub pull requests (PRs).

To create a new PR, fork the project in GitHub and clone the upstream repo:

```sh
$ go get -d github.com/observiq/stanza
```

This will put the project in `${GOPATH}/src/github.com/observiq/stanza`. You can alternatively use `git` directly with:

```sh
$ git clone https://github.com/observiq/stanza
```

This would put the project in the `stanza` directory in current working directory.

Enter the newly created directory and add your fork as a new remote:

```sh
$ git remote add <YOUR_FORK> git@github.com:<YOUR_GITHUB_USERNAME>/opentelemetry-go
```

Check out a new branch, make modifications, run linters and tests, and push the branch to your fork:

```sh
$ git checkout -b <YOUR_BRANCH_NAME>
# edit files
$ go test -race ./...
$ git add -p
$ git commit
$ git push <YOUR_FORK> <YOUR_BRANCH_NAME>
```

Open a pull request against the main `stanza` repo.


### How to Receive Comments

* If the PR is not ready for review, please put `[WIP]` in the title, tag it as `work-in-progress`, or mark it as [`draft`](https://github.blog/2019-02-14-introducing-draft-pull-requests/).
* If you're stuck, tag a maintainer and ask a question. We're here to help each other.
* Make sure CI passes.


### How to Get PRs Merged

A PR is considered to be **ready to merge** when:

* It has received approval from at least one maintainer.
* CI passes.
* Major feedback is resolved.


## Design Choices

Best practices for developing a builtin operator are documented below, but for changes to the core agent, we are happy to discuss proposals in the issue tracker.


### Operator Development

In order to write a builtin operator, follow these three steps:
1. Build a unique struct that satisfies the [`Operator`](operator/operator.go) interface. This struct will define what your operator does when executed in the pipeline.

```go
type ExampleOperator struct {
	FilePath string
}

func (p *ExampleOperator) Process(ctx context.Context, entry *entry.Entry) error {
	// Processing logic
}
```

2. Build a unique config struct that satisfies the [`Config`](operator/config.go) interface. This struct will define the parameters used to configure and build your operator struct in step 1.

```go
type ExampleOperatorConfig struct {
	filePath string
}

func (c ExampleOperatorConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	return &ExampleOperator{
		filePath: c.FilePath,
	}, nil
}
```

3. Register your config struct in the operator registry using an `init()` hook. This will ensure that the agent knows about your operator at runtime and can build it from a YAML config.

```go
func init() {
	operator.RegisterOperator("example_operator", &ExampleOperatorConfig{})
}
```


#### Any tips for building operators?
We highly recommend that developers take advantage of [helpers](operator/helper) when building their operators. Helpers are structs that help satisfy common behavior shared across many operators. By embedding these structs, you can skip having to satisfy certain aspects of the `operator` and `config` interfaces.

For example, almost all operators should embed the [BasicOperator](operator/helper/operator.go) helper, as it provides simple functionality for returning an operator id and operator type.

```go
// ExampleOperator is a basic operator, with a basic lifecycle, that consumes
// but doesn't send log entries. Rather than implementing every part of the operator
// interface, we can embed the following helpers to achieve this effect.
type ExampleOperator struct {
	helper.BasicOperator
	helper.BasicLifecycle
	helper.BasicOutput
}
```
