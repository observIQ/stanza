package netflow

import (
	"encoding/json"
	"flag"
	"fmt"
	"runtime"
	"sync"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/cloudflare/goflow/v3/utils"
	log "github.com/sirupsen/logrus"
)

var (
	version    = ""
	buildinfos = ""
	AppVersion = "GoFlow " + version + " " + buildinfos

	SFlowEnable = flag.Bool("sflow", true, "Enable sFlow")
	SFlowAddr   = flag.String("sflow.addr", "", "sFlow listening address")
	SFlowPort   = flag.Int("sflow.port", 6343, "sFlow listening port")
	SFlowReuse  = flag.Bool("sflow.reuserport", false, "Enable so_reuseport for sFlow")

	NFLEnable = flag.Bool("nfl", true, "Enable NetFlow v5")
	NFLAddr   = flag.String("nfl.addr", "", "NetFlow v5 listening address")
	NFLPort   = flag.Int("nfl.port", 2056, "NetFlow v5 listening port")
	NFLReuse  = flag.Bool("nfl.reuserport", false, "Enable so_reuseport for NetFlow v5")

	NFEnable = flag.Bool("nf", true, "Enable NetFlow/IPFIX")
	NFAddr   = flag.String("nf.addr", "", "NetFlow/IPFIX listening address")
	NFPort   = flag.Int("nf.port", 2055, "NetFlow/IPFIX listening port")
	NFReuse  = flag.Bool("nf.reuserport", false, "Enable so_reuseport for NetFlow/IPFIX")

	Workers  = flag.Int("workers", 1, "Number of workers per collector")
	LogLevel = flag.String("loglevel", "info", "Log level")
	LogFmt   = flag.String("logfmt", "normal", "Log formatter")

	EnableKafka = flag.Bool("kafka", true, "Enable Kafka")
	FixedLength = flag.Bool("proto.fixedlen", false, "Enable fixed length protobuf")
	MetricsAddr = flag.String("metrics.addr", ":8080", "Metrics address")
	MetricsPath = flag.String("metrics.path", "/metrics", "Metrics path")

	TemplatePath = flag.String("templates.path", "/templates", "NetFlow/IPFIX templates list")

	Version = flag.Bool("v", false, "Print version")
)

// Transport Implementation
type Transport struct {
}

// Initialize is required by GoFlows util.Transport interface
func (t *Transport) Initialize() {

}

// Publish is required by GoFlows util.Transport interface
func (t Transport) Publish(messages []*flowmessage.FlowMessage) {
	t.Initialize() // TODO: probably should go somewhere else

	go func() {
		for _, msg := range messages {
			m, err := ToMapStringInterface(msg)
			if err != nil {
				log.Error(err)
				continue
			}

			b, err := json.Marshal(m)
			if err != nil {
				log.Error(err)
				continue
			}
			fmt.Println(string(b))

			// Process messages
			// Putting the messages in a channel without forking a new thread instead would also work.
		}
	}()

}

// more or less copied from goflows main package
func startGoFlow(transport utils.Transport) {

	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Info("Starting GoFlow")

	sSFlow := &utils.StateSFlow{
		Transport: transport,
		Logger:    log.StandardLogger(),
	}
	sNF := &utils.StateNetFlow{
		Transport: transport,
		Logger:    log.StandardLogger(),
	}
	sNFL := &utils.StateNFLegacy{
		Transport: transport,
		Logger:    log.StandardLogger(),
	}

	// TODO: This will expose prom metrics, do we want this?
	//go httpServer(sNF)

	wg := &sync.WaitGroup{}
	if *SFlowEnable {
		wg.Add(1)
		go func() {
			log.WithFields(log.Fields{
				"Type": "sFlow"}).
				Infof("Listening on UDP %v:%v", *SFlowAddr, *SFlowPort)

			err := sSFlow.FlowRoutine(*Workers, *SFlowAddr, *SFlowPort, *SFlowReuse)
			if err != nil {
				log.Fatalf("Fatal error: could not listen to UDP (%v)", err)
			}
			wg.Done()
		}()
	}
	if *NFEnable {
		wg.Add(1)
		go func() {
			log.WithFields(log.Fields{
				"Type": "NetFlow"}).
				Infof("Listening on UDP %v:%v", *NFAddr, *NFPort)

			err := sNF.FlowRoutine(*Workers, *NFAddr, *NFPort, *NFReuse)
			if err != nil {
				log.Fatalf("Fatal error: could not listen to UDP (%v)", err)
			}
			wg.Done()
		}()
	}
	if *NFLEnable {
		wg.Add(1)
		go func() {
			log.WithFields(log.Fields{
				"Type": "NetFlowLegacy"}).
				Infof("Listening on UDP %v:%v", *NFLAddr, *NFLPort)

			err := sNFL.FlowRoutine(*Workers, *NFLAddr, *NFLPort, *NFLReuse)
			if err != nil {
				log.Fatalf("Fatal error: could not listen to UDP (%v)", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
