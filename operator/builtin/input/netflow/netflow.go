package netflow

import (
	"context"
	"flag"
	"fmt"
	"net"
	"runtime"
	"sync"

	flowmessage "github.com/cloudflare/goflow/v3/pb"
	"github.com/cloudflare/goflow/v3/utils"
	"github.com/fatih/structs"
	"github.com/observiq/stanza/errors"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

var (
	version    = ""
	buildinfos = ""
	AppVersion = "GoFlow " + version + " " + buildinfos

	SFlowEnable = flag.Bool("sflow", true, "Enable sFlow")
	SFlowAddr   = flag.String("sflow.addr", "", "sFlow listening address")
	SFlowPort   = flag.Int("sflow.port", 6343, "sFlow listening port")
	SFlowReuse  = flag.Bool("sflow.reuserport", false, "Enable so_reuseport for sFlow")

	/*NFLEnable = flag.Bool("nfl", true, "Enable NetFlow v5")
	NFLAddr   = flag.String("nfl.addr", "", "NetFlow v5 listening address")
	NFLPort   = flag.Int("nfl.port", 2056, "NetFlow v5 listening port")
	NFLReuse  = flag.Bool("nfl.reuserport", false, "Enable so_reuseport for NetFlow v5")*/

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

// Publish is required by GoFlows util.Transport interface
func (n NetflowInput) Publish(messages []*flowmessage.FlowMessage) {
	go func() {
		for _, msg := range messages {
			structParser := structs.New(msg)
			structParser.TagName = "json"
			m := structParser.Map()

			// https://github.com/cloudflare/goflow/blob/ddd88a7faa89bd9a8e75f0ceca17cbb443c14a8f/pb/flow.pb.go#L57
			// IP address keys are []byte encoded
			byteKeys := [...]string{
				"SamplerAddress",
				"SrcAddr",
				"DstAddr",
				"NextHop",
				"SrcAddrEncap",
				"DstAddrEncap",
			}
			var err error
			for _, key := range byteKeys {
				m, err = mapBytesToString(m, key)
				if err != nil {
					n.Errorf(fmt.Sprintf("error converting %s to string", key), zap.Error(err))
				}

			}

			entry, err := n.NewEntry(m)
			if err != nil {
				log.Error(err)
				continue
			}
			n.Write(context.Background(), entry)
		}
	}()

}

// more or less copied from goflows main package
func startGoFlow(transport utils.Transport, NFLEnable, NFLReuse bool, NFLAddr string, NFLPort int) {

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
	if NFLEnable {
		wg.Add(1)
		go func() {
			log.WithFields(log.Fields{
				"Type": "NetFlowLegacy"}).
				Infof("Listening on UDP %v:%v", NFLAddr, NFLPort)

			err := sNFL.FlowRoutine(*Workers, NFLAddr, NFLPort, NFLReuse)
			if err != nil {
				log.Fatalf("Fatal error: could not listen to UDP (%v)", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

// converts a key from []byte to string if it exists
func mapBytesToString(m map[string]interface{}, key string) (map[string]interface{}, error) {
	if val, ok := m[key]; ok {
		delete(m, key)
		switch x := val.(type) {
		case []byte:
			ip, err := bytesToIP(x)
			if err != nil {
				return nil, errors.Wrap(err, "error converting DstAddr to string")
			}
			m[key] = ip.String()
			return m, nil
		default:
			return nil, fmt.Errorf("type %T cannot be parsed as an IP address", val)
		}

	}
	return m, nil
}

func bytesToIP(b []byte) (net.IP, error) {
	switch x := len(b); x {
	case 4, 16:
		var ip net.IP = b
		return ip, nil
	default:
		return nil, fmt.Errorf("cannot convert byte slice to ip address, expected length of 4 or 16 got %d", x)
	}
}
