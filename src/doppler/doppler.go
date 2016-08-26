package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	doppler_config "doppler/config"
	"doppler/pb"
	"doppler/sinkserver"
	"doppler/sinkserver/blacklist"
	"doppler/sinkserver/sinkmanager"
	"doppler/sinkserver/websocketserver"

	"doppler/listeners"
	"monitor"

	"github.com/cloudfoundry/dropsonde"
	"github.com/cloudfoundry/dropsonde/dropsonde_unmarshaller"
	"github.com/cloudfoundry/dropsonde/metric_sender"
	"github.com/cloudfoundry/dropsonde/metricbatcher"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/dropsonde/signature"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/appservice"
	"github.com/cloudfoundry/loggregatorlib/store"
	"github.com/cloudfoundry/loggregatorlib/store/cache"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/cloudfoundry/storeadapter"
	"github.com/gogo/protobuf/proto"
)

type Doppler struct {
	*gosteno.Logger
	batcher *metricbatcher.MetricBatcher

	appStoreWatcher *store.AppServiceStoreWatcher

	errChan         chan error
	udpListener     *listeners.UDPListener
	tcpListener     *listeners.TCPListener
	tlsListener     *listeners.TCPListener
	sinkManager     *sinkmanager.SinkManager
	messageRouter   *sinkserver.MessageRouter
	websocketServer *websocketserver.WebsocketServer

	dropsondeUnmarshallerCollection *dropsonde_unmarshaller.DropsondeUnmarshallerCollection
	dropsondeBytesChan              <-chan []byte
	dropsondeVerifiedBytesChan      chan []byte
	envelopeChan                    chan *events.Envelope
	signatureVerifier               *signature.Verifier

	storeAdapter storeadapter.StoreAdapter

	uptimeMonitor   *monitor.Uptime
	openFileMonitor *monitor.LinuxFileDescriptor

	newAppServiceChan, deletedAppServiceChan <-chan appservice.AppService
	wg                                       sync.WaitGroup
}

func New(
	logger *gosteno.Logger,
	host string,
	config *doppler_config.Config,
	storeAdapter storeadapter.StoreAdapter,
	messageDrainBufferSize uint,
	dropsondeOrigin string,
	websocketWriteTimeout time.Duration,
	dialTimeout time.Duration,
) (*Doppler, error) {
	doppler := &Doppler{
		Logger:                     logger,
		storeAdapter:               storeAdapter,
		dropsondeVerifiedBytesChan: make(chan []byte),
	}

	keepAliveInterval := 30 * time.Second

	appStoreCache := cache.NewAppServiceCache()
	doppler.appStoreWatcher, doppler.newAppServiceChan, doppler.deletedAppServiceChan = store.NewAppServiceStoreWatcher(storeAdapter, appStoreCache, logger)

	doppler.batcher = initializeMetrics(config.MetricBatchIntervalMilliseconds)

	doppler.envelopeChan = make(chan *events.Envelope)

	doppler.udpListener, doppler.dropsondeBytesChan = listeners.NewUDPListener(
		fmt.Sprintf("%s:%d", host, config.IncomingUDPPort),
		doppler.batcher,
		logger,
		"udpListener",
	)

	var err error
	if config.EnableTLSTransport {
		tlsConfig := &config.TLSListenerConfig
		addr := fmt.Sprintf("%s:%d", host, tlsConfig.Port)
		contextName := "tlsListener"
		doppler.tlsListener, err = listeners.NewTCPListener(contextName, addr, tlsConfig, doppler.envelopeChan, doppler.batcher, logger)
		if err != nil {
			return nil, err
		}
	}

	addr := fmt.Sprintf("%s:%d", host, config.IncomingTCPPort)
	contextName := "tcpListener"
	doppler.tcpListener, err = listeners.NewTCPListener(contextName, addr, nil, doppler.envelopeChan, doppler.batcher, logger)

	doppler.signatureVerifier = signature.NewVerifier(logger, config.SharedSecret)

	doppler.dropsondeUnmarshallerCollection = dropsonde_unmarshaller.NewDropsondeUnmarshallerCollection(logger, config.UnmarshallerCount)

	blacklist := blacklist.New(config.BlackListIps, logger)
	metricTTL := time.Duration(config.ContainerMetricTTLSeconds) * time.Second
	sinkTimeout := time.Duration(config.SinkInactivityTimeoutSeconds) * time.Second
	sinkIOTimeout := time.Duration(config.SinkIOTimeoutSeconds) * time.Second
	doppler.sinkManager = sinkmanager.New(
		config.MaxRetainedLogMessages,
		config.SinkSkipCertVerify,
		blacklist,
		logger,
		messageDrainBufferSize,
		dropsondeOrigin,
		sinkTimeout,
		sinkIOTimeout,
		metricTTL,
		dialTimeout,
	)
	doppler.messageRouter = sinkserver.NewMessageRouter(doppler.sinkManager, logger)

	doppler.websocketServer, err = websocketserver.New(
		fmt.Sprintf(":%d", config.OutgoingPort),
		doppler.sinkManager,
		websocketWriteTimeout,
		keepAliveInterval,
		config.MessageDrainBufferSize,
		dropsondeOrigin,
		doppler.batcher,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the websocket server: %s", err.Error())
	}

	monitorInterval := time.Duration(config.MonitorIntervalSeconds) * time.Second
	doppler.openFileMonitor = monitor.NewLinuxFD(monitorInterval, logger)
	doppler.uptimeMonitor = monitor.NewUptime(monitorInterval)

	return doppler, nil
}

func (doppler *Doppler) Subscribe(_ *pb.Subscription, TC pb.Doppler_SubscribeServer) error {
	log.Print("New subscription...")
	for e := range doppler.envelopeChan {
		data, _ := proto.Marshal(e)
		if err := TC.Send(&pb.DataPacket{data}); err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}

func (doppler *Doppler) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9999))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDopplerServer(s, doppler)
	go s.Serve(lis)

	doppler.errChan = make(chan error)

	doppler.wg.Add(7 + doppler.dropsondeUnmarshallerCollection.Size())

	go func() {
		defer doppler.wg.Done()
		doppler.appStoreWatcher.Run()
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.udpListener.Start()
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.tcpListener.Start()
	}()

	if doppler.tlsListener != nil {
		go func() {
			doppler.wg.Add(1)
			defer doppler.wg.Done()
			doppler.tlsListener.Start()
		}()
	}

	udpEnvelopes := make(chan *events.Envelope)
	doppler.dropsondeUnmarshallerCollection.Run(doppler.dropsondeVerifiedBytesChan, udpEnvelopes, &doppler.wg)
	go func() {
		for {
			env := <-udpEnvelopes
			doppler.batcher.BatchCounter("listeners.receivedEnvelopes").
				SetTag("protocol", "udp").
				SetTag("event_type", env.GetEventType().String()).
				Increment()
			doppler.envelopeChan <- env
		}
	}()

	go func() {
		defer func() {
			doppler.wg.Done()
			close(doppler.dropsondeVerifiedBytesChan)
		}()
		doppler.signatureVerifier.Run(doppler.dropsondeBytesChan, doppler.dropsondeVerifiedBytesChan)
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.sinkManager.Start(doppler.newAppServiceChan, doppler.deletedAppServiceChan)
	}()

	go func() {
		defer func() {
			doppler.wg.Done()
			close(doppler.envelopeChan)
		}()
		doppler.messageRouter.Start(doppler.envelopeChan)
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.websocketServer.Start()
	}()

	go doppler.uptimeMonitor.Start()
	go doppler.openFileMonitor.Start()

	// The following runs forever. Put all startup functions above here.
	for err := range doppler.errChan {
		doppler.Errorf("Got error %s", err)
	}
}

func (doppler *Doppler) Stop() {
	go doppler.udpListener.Stop()
	go doppler.tcpListener.Stop()
	go doppler.tlsListener.Stop()
	go doppler.sinkManager.Stop()
	go doppler.messageRouter.Stop()
	go doppler.websocketServer.Stop()
	doppler.appStoreWatcher.Stop()
	doppler.wg.Wait()

	doppler.storeAdapter.Disconnect()
	close(doppler.errChan)
	doppler.uptimeMonitor.Stop()
	doppler.openFileMonitor.Stop()
}

func initializeMetrics(batchIntervalMilliseconds uint) *metricbatcher.MetricBatcher {
	eventEmitter := dropsonde.AutowiredEmitter()
	metricSender := metric_sender.NewMetricSender(eventEmitter)
	metricBatcher := metricbatcher.New(metricSender, time.Duration(batchIntervalMilliseconds)*time.Millisecond)
	metrics.Initialize(metricSender, metricBatcher)
	return metricBatcher
}
