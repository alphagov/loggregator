package dopplerproxy

import (
	"doppler/pb"
	"log"
	"net/http"
	"trafficcontroller/authorization"
	"trafficcontroller/doppler_endpoint"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/cloudfoundry/gosteno"
	"github.com/gorilla/websocket"
)

const FIREHOSE_ID = "firehose"

type Proxy struct {
	logAuthorize   authorization.LogAccessAuthorizer
	adminAuthorize authorization.AdminAccessAuthorizer
	connector      channelGroupConnector
	translate      RequestTranslator
	cookieDomain   string
	logger         *gosteno.Logger

	grpcConn pb.DopplerClient
	upgrader websocket.Upgrader
}

type RequestTranslator func(request *http.Request) (*http.Request, error)

type channelGroupConnector interface {
	Connect(dopplerEndpoint doppler_endpoint.DopplerEndpoint, messagesChan chan<- []byte, stopChan <-chan struct{})
}

func NewDopplerProxy(logAuthorize authorization.LogAccessAuthorizer, adminAuthorizer authorization.AdminAccessAuthorizer, connector channelGroupConnector, translator RequestTranslator, cookieDomain string, logger *gosteno.Logger) *Proxy {
	conn, err := grpc.Dial("10.244.0.134:9999", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	return &Proxy{
		logAuthorize:   logAuthorize,
		adminAuthorize: adminAuthorizer,
		connector:      connector,
		translate:      translator,
		cookieDomain:   cookieDomain,
		logger:         logger,

		grpcConn: pb.NewDopplerClient(conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (proxy *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	subscription := new(pb.Subscription)
	reader, err := proxy.grpcConn.Subscribe(context.Background(), subscription)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := proxy.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		data, err := reader.Recv()
		log.Print("GOT DATA FROM GRPC!!")
		if err != nil {
			log.Print(err)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data.Message); err != nil {
			log.Print(err)
			return
		}
	}
}

type TrafficControllerMonitor struct {
}

func (hm TrafficControllerMonitor) Ok() bool {
	return true
}
