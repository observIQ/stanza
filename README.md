# Bindplane Log Agent
## Commands
To run: `go run ./cmd --config ./dev/config.yml`

To test: `go test -cover -race ./...` (cover and race optional)

To benchmark: `go test -bench=. -test.benchmem=true ./...`

## Plugin Development
Plugins should be designed to executed only a single responsibility, such as:
- Discovering logs
- Filtering logs
- Parsing logs
- Forwarding logs

### Basic Methods
Plugins implement 4 basic methods: 
- **ID()**
  - This method should return the plugin id specified in the plugin config.
  - We highly recommend embedding a basic plugin helper to implement this method.
- **Type()**
  - This method should return the plugin type specified in the plugin config.
  - We highly recommend embedding a basic plugin helper to implement this method.
- **Start()**
  - This method should start the execution of a plugin's responsibility.
  - This method should never block. Doing so will block other plugins in the pipeline from starting. If a plugin has long running behavior, like discovering logs, we recommend starting a goroutine from this method.
  - This method should only return an error if a clear and present obstruction exists that prevents the plugin from starting. In the case of plugins with long running behavior, where an obstruction may occur in the future, it is the plugin's responsibility to log these errors.
  - If a plugin does not require any setup before starting, we recommend embedding a basic plugin helper to handle this method.
- **Stop()**
  - This method should stop the execution of a plugin's responsibility.
  - This method should block until the plugin has completely stopped. This will prevent plugins further in the pipeline from stopping should they receive further input from this plugin.
  - This method should return an error if something goes wrong when stopping the plugin.
  - If a plugin does not require any cleanup before stopping, we recommend embedding a basic plugin helper to handle this method.
