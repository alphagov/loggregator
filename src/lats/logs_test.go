package lats_test

import (
	"crypto/tls"
	"fmt"
	"lats/helpers"
	"time"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logs", func() {
	It("Displays recent logs", func() {
		var envelope *events.Envelope
		for i := 0; i < 10; i++ {
			envelope = testLogMessage(fmt.Sprintf("testMessage %d", i))
			helpers.EmitToMetron(envelope)
		}

		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		consumer := consumer.New(config.TrafficcontrollerURL, tlsConfig, nil)

		token := helpers.GetAuthToken()
		_, err := consumer.RecentLogs("lats-test", token)
		Expect(err).NotTo(HaveOccurred())
	})
})

func testLogMessage(msg string) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String("lats"),
		EventType: events.Envelope_LogMessage.Enum(),
		LogMessage: &events.LogMessage{
			Message:     []byte(msg),
			MessageType: events.LogMessage_OUT.Enum(),
			Timestamp:   proto.Int64(time.Now().UnixNano()),
			AppId:       proto.String("lats-test"),
		},
	}
}
