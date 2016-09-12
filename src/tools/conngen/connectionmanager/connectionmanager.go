package connectionmanager

import (
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"os"
	"tools/conngen/config"

	"github.com/cloudfoundry/noaa/consumer"
)

type ConnectionManager struct {
	consumers []*consumer.Consumer
	conf      *config.Config
}

func New(conf *config.Config) *ConnectionManager {
	var consumers []*consumer.Consumer
	for _, tcAddress := range conf.TCAddresses {
		c := consumer.New(tcAddress, &tls.Config{InsecureSkipVerify: true}, nil)
		consumers = append(consumers, c)
	}

	return &ConnectionManager{
		conf:      conf,
		consumers: consumers,
	}
}

func (c *ConnectionManager) pick() *consumer.Consumer {
	pos := rand.Intn(len(c.consumers))
	return c.consumers[pos]
}

func (c *ConnectionManager) NewFirehose() {
	log.Print("Creating New Firehose Connection")
	consumer := c.pick()
	_, errorChan := consumer.FirehoseWithoutReconnect(c.conf.SubscriptionId, c.conf.AuthToken)
	for err := range errorChan {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
	}
}

func (c *ConnectionManager) NewStream() {
	log.Print("Creating New Stream Connection")
	consumer := c.pick()
	_, errorChan := consumer.StreamWithoutReconnect(c.conf.SubscriptionId, c.conf.AuthToken)
	for err := range errorChan {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
	}
}

func (c *ConnectionManager) Close() {
	for _, consumer := range c.consumers {
		consumer.Close()
	}
}
