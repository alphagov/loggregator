package connectionmanager_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"
	"tools/conngen/config"
	"tools/conngen/connectionmanager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gorilla/websocket"
)

var _ = Describe("Connection", func() {
	Describe("NewFirehose", func() {
		It("creates a connection to the firehose endpoint", func() {
			handler := &tcServer{}
			server := httptest.NewServer(handler)
			defer server.Close()

			conf := &config.Config{
				TCAddresses: []string{strings.Replace(server.URL, "http", "ws", 1)},
			}
			conn := connectionmanager.New(conf)
			conn.NewFirehose()

			f := func() int64 {
				return handler.FirehoseCount()
			}
			Eventually(f).Should(BeEquivalentTo(1))
		})
	})

	Describe("NewStream", func() {
		It("creates a connection to the stream endpoint", func() {
			handler := &tcServer{}
			server := httptest.NewServer(handler)
			defer server.Close()

			conf := &config.Config{
				TCAddresses: []string{strings.Replace(server.URL, "http", "ws", 1)},
			}
			conn := connectionmanager.New(conf)
			conn.NewStream()

			f := func() int64 {
				return handler.StreamCount()
			}
			Eventually(f).Should(BeEquivalentTo(1))
		})
	})
})

type tcServer struct {
	streamCount   int64
	firehoseCount int64
}

func (tc *tcServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		println(err.Error())
	}

	defer func() {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
		ws.Close()
	}()

	if strings.Contains(r.URL.Path, "stream") {
		atomic.AddInt64(&tc.streamCount, 1)
	} else if strings.Contains(r.URL.Path, "firehose") {
		atomic.AddInt64(&tc.firehoseCount, 1)
	}
}

func (tc *tcServer) StreamCount() int64 {
	return atomic.LoadInt64(&tc.streamCount)
}

func (tc *tcServer) FirehoseCount() int64 {
	return atomic.LoadInt64(&tc.firehoseCount)
}
