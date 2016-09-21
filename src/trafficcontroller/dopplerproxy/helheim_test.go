// This file was generated by github.com/nelsam/hel.  Do not
// edit this code by hand unless you *really* know what you're
// doing.  Expect any changes made manually to be overwritten
// the next time hel regenerates this file.

package dopplerproxy_test

import (
	"plumbing"
	"time"
	"trafficcontroller/doppler_endpoint"
	"trafficcontroller/grpcconnector"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type mockGrpcConnector struct {
	StreamCalled chan bool
	StreamInput  struct {
		Ctx  chan context.Context
		In   chan *plumbing.StreamRequest
		Opts chan []grpc.CallOption
	}
	StreamOutput struct {
		Ret0 chan grpcconnector.Receiver
		Ret1 chan error
	}
	FirehoseCalled chan bool
	FirehoseInput  struct {
		Ctx  chan context.Context
		In   chan *plumbing.FirehoseRequest
		Opts chan []grpc.CallOption
	}
	FirehoseOutput struct {
		Ret0 chan grpcconnector.Receiver
		Ret1 chan error
	}
}

func newMockGrpcConnector() *mockGrpcConnector {
	m := &mockGrpcConnector{}
	m.StreamCalled = make(chan bool, 100)
	m.StreamInput.Ctx = make(chan context.Context, 100)
	m.StreamInput.In = make(chan *plumbing.StreamRequest, 100)
	m.StreamInput.Opts = make(chan []grpc.CallOption, 100)
	m.StreamOutput.Ret0 = make(chan grpcconnector.Receiver, 100)
	m.StreamOutput.Ret1 = make(chan error, 100)
	m.FirehoseCalled = make(chan bool, 100)
	m.FirehoseInput.Ctx = make(chan context.Context, 100)
	m.FirehoseInput.In = make(chan *plumbing.FirehoseRequest, 100)
	m.FirehoseInput.Opts = make(chan []grpc.CallOption, 100)
	m.FirehoseOutput.Ret0 = make(chan grpcconnector.Receiver, 100)
	m.FirehoseOutput.Ret1 = make(chan error, 100)
	return m
}
func (m *mockGrpcConnector) Stream(ctx context.Context, in *plumbing.StreamRequest, opts ...grpc.CallOption) (grpcconnector.Receiver, error) {
	m.StreamCalled <- true
	m.StreamInput.Ctx <- ctx
	m.StreamInput.In <- in
	m.StreamInput.Opts <- opts
	return <-m.StreamOutput.Ret0, <-m.StreamOutput.Ret1
}
func (m *mockGrpcConnector) Firehose(ctx context.Context, in *plumbing.FirehoseRequest, opts ...grpc.CallOption) (grpcconnector.Receiver, error) {
	m.FirehoseCalled <- true
	m.FirehoseInput.Ctx <- ctx
	m.FirehoseInput.In <- in
	m.FirehoseInput.Opts <- opts
	return <-m.FirehoseOutput.Ret0, <-m.FirehoseOutput.Ret1
}

type mockChannelGroupConnector struct {
	ConnectCalled chan bool
	ConnectInput  struct {
		DopplerEndpoint chan doppler_endpoint.DopplerEndpoint
		MessagesChan    chan (chan<- []byte)
		StopChan        chan (<-chan struct{})
	}
}

func newMockChannelGroupConnector() *mockChannelGroupConnector {
	m := &mockChannelGroupConnector{}
	m.ConnectCalled = make(chan bool, 100)
	m.ConnectInput.DopplerEndpoint = make(chan doppler_endpoint.DopplerEndpoint, 100)
	m.ConnectInput.MessagesChan = make(chan (chan<- []byte), 100)
	m.ConnectInput.StopChan = make(chan (<-chan struct{}), 100)
	return m
}
func (m *mockChannelGroupConnector) Connect(dopplerEndpoint doppler_endpoint.DopplerEndpoint, messagesChan chan<- []byte, stopChan <-chan struct{}) {
	m.ConnectCalled <- true
	m.ConnectInput.DopplerEndpoint <- dopplerEndpoint
	m.ConnectInput.MessagesChan <- messagesChan
	m.ConnectInput.StopChan <- stopChan
}

type mockContext struct {
	DeadlineCalled chan bool
	DeadlineOutput struct {
		Deadline chan time.Time
		Ok       chan bool
	}
	DoneCalled chan bool
	DoneOutput struct {
		Ret0 chan (<-chan struct{})
	}
	ErrCalled chan bool
	ErrOutput struct {
		Ret0 chan error
	}
	ValueCalled chan bool
	ValueInput  struct {
		Key chan interface{}
	}
	ValueOutput struct {
		Ret0 chan interface{}
	}
}

func newMockContext() *mockContext {
	m := &mockContext{}
	m.DeadlineCalled = make(chan bool, 100)
	m.DeadlineOutput.Deadline = make(chan time.Time, 100)
	m.DeadlineOutput.Ok = make(chan bool, 100)
	m.DoneCalled = make(chan bool, 100)
	m.DoneOutput.Ret0 = make(chan (<-chan struct{}), 100)
	m.ErrCalled = make(chan bool, 100)
	m.ErrOutput.Ret0 = make(chan error, 100)
	m.ValueCalled = make(chan bool, 100)
	m.ValueInput.Key = make(chan interface{}, 100)
	m.ValueOutput.Ret0 = make(chan interface{}, 100)
	return m
}
func (m *mockContext) Deadline() (deadline time.Time, ok bool) {
	m.DeadlineCalled <- true
	return <-m.DeadlineOutput.Deadline, <-m.DeadlineOutput.Ok
}
func (m *mockContext) Done() <-chan struct{} {
	m.DoneCalled <- true
	return <-m.DoneOutput.Ret0
}
func (m *mockContext) Err() error {
	m.ErrCalled <- true
	return <-m.ErrOutput.Ret0
}
func (m *mockContext) Value(key interface{}) interface{} {
	m.ValueCalled <- true
	m.ValueInput.Key <- key
	return <-m.ValueOutput.Ret0
}

type mockReceiver struct {
	RecvCalled chan bool
	RecvOutput struct {
		Ret0 chan *plumbing.Response
		Ret1 chan error
	}
}

func newMockReceiver() *mockReceiver {
	m := &mockReceiver{}
	m.RecvCalled = make(chan bool, 100)
	m.RecvOutput.Ret0 = make(chan *plumbing.Response, 100)
	m.RecvOutput.Ret1 = make(chan error, 100)
	return m
}
func (m *mockReceiver) Recv() (*plumbing.Response, error) {
	m.RecvCalled <- true
	return <-m.RecvOutput.Ret0, <-m.RecvOutput.Ret1
}
