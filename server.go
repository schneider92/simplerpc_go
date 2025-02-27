package simplerpc

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Service interface that each service has to implement. Developer has nothing
// to do with this as this implementation is automatically generated
type ServerService interface {
	GetServiceId() int64
	GetRevision() string
	CallFunction(ctx context.Context, functionId int64, requestBytes []byte, respBytes []byte) []byte
}

type canceller struct {
	mu      sync.Mutex
	cancels map[int64]context.CancelFunc
}

func (c *canceller) addRequest(ctx context.Context, requestId int64) context.Context {
	// create context with cancellation
	ctx, cancel := context.WithCancel(ctx)

	// lock mutex
	c.mu.Lock()
	defer c.mu.Unlock()

	// add cancellation
	c.cancels[requestId] = cancel

	// done
	return ctx
}

func (c *canceller) requestFinished(requestId int64) (cancelled bool) {
	// lock mutex
	c.mu.Lock()
	defer c.mu.Unlock()

	// find cancel
	_, found := c.cancels[requestId]

	// if found, delete it
	if found {
		delete(c.cancels, requestId)
	}

	// return cancelled if not found
	return !found
}

func (c *canceller) cancelRequest(requestId int64) (found bool) {
	// lock mutex
	c.mu.Lock()
	defer c.mu.Unlock()

	// find cancel
	cancel, found := c.cancels[requestId]

	// if found, delete it and cancel
	if found {
		delete(c.cancels, requestId)
		cancel()
	}

	// done
	return found
}

// Server type wrapping the services
type Server struct {
	canceller *canceller
	services  []ServerService
}

// Create a new server with the given services. Return error if any service has invalid id
func NewServer(services []ServerService) (srv Server, err error) {
	// validate services
	for curr := range services {
		// check if id is valid
		id := services[curr].GetServiceId()
		if id <= 0 {
			err = fmt.Errorf("could not create server: service at index %d has invalid id", id)
			return
		}

		// check if id is not repeating
		for i := 0; i < curr; i++ {
			if id == services[i].GetServiceId() {
				err = fmt.Errorf("could not create server: services at indices %d and %d has the same id: %d", i, curr, id)
				return
			}
		}
	}

	// return server instance and no error
	srv.canceller = &canceller{
		cancels: map[int64]context.CancelFunc{},
	}
	srv.services = services
	return
}

func (srv Server) handleServerRequestGetServices(respBytes []byte) []byte {
	// write array length
	respBytes = SerializeInteger(respBytes, int64(len(srv.services)))

	// write each elements
	for _, service := range srv.services {
		respBytes = SerializeInteger(respBytes, service.GetServiceId())
		respBytes = SerializeString(respBytes, service.GetRevision())
	}

	// done
	return respBytes
}

func (srv Server) handleServerRequestCancel(requestBytes []byte) {
	// read request id to cancel
	requestBytes, requestIdToCancel := DeserializeInteger(requestBytes)
	if requestBytes != nil {
		srv.canceller.cancelRequest(requestIdToCancel)
	}
}

func (srv Server) handleServerRequestEcho(requestBytes []byte, respBytes []byte) []byte {
	// read wait time
	requestBytes, wait_ms := DeserializeInteger(requestBytes)

	// wait that much time
	if wait_ms > 0 {
		time.Sleep(time.Duration(int64(time.Millisecond) * wait_ms))
	}

	// copy request bytes
	return append(respBytes, requestBytes...)
}

func (srv Server) callFunctionOnServer(functionId int64, requestBytes []byte, respBytes []byte) []byte {
	// get services
	if functionId == 0 {
		return srv.handleServerRequestGetServices(respBytes)
	}

	// cancel request
	if functionId == 1 {
		srv.handleServerRequestCancel(requestBytes)
		return respBytes
	}

	// echo
	if functionId == 2 {
		return srv.handleServerRequestEcho(requestBytes, respBytes)
	}

	// unknown func (or nop, for which we also do nothing)
	return nil
}

func (srv Server) callFunctionOnService(ctx context.Context, service ServerService, requestId, functionId int64, requestBytes []byte, respBytes []byte) []byte {
	// set up cancellation
	ctx = srv.canceller.addRequest(ctx, requestId)

	// call the function
	respBytes = service.CallFunction(ctx, functionId, requestBytes, respBytes)

	// finish cancellation
	cancelled := srv.canceller.requestFinished(requestId)

	// done
	if cancelled {
		return nil
	} else {
		return respBytes
	}
}

func (srv Server) handleService(ctx context.Context, requestId, serviceId, functionId int64, requestBytes []byte, respBytes []byte) []byte {
	// if service id is 0, this request is server-related and we need to handle it here
	if serviceId == 0 {
		return srv.callFunctionOnServer(functionId, requestBytes, respBytes)
	}

	// find service
	for _, service := range srv.services {
		if service.GetServiceId() == serviceId {
			return srv.callFunctionOnService(ctx, service, requestId, functionId, requestBytes, respBytes)
		}
	}

	// not found
	return nil
}

// Process a request represented by the given bytes. On success, the response is
// appended to respBytes and is returned.
func (srv Server) ProcessRequest(ctx context.Context, requestBytes []byte, respBytes []byte) []byte {
	// parse headers
	requestBytes, requestId := DeserializeInteger(requestBytes)
	requestBytes, serviceId := DeserializeInteger(requestBytes)
	requestBytes, functionId := DeserializeInteger(requestBytes)
	if requestBytes == nil {
		return nil
	}

	// prepare result buffer
	originalResp := respBytes
	if requestId > 0 {
		// write sequence id and integer 1 (=success)
		respBytes = SerializeInteger(respBytes, requestId)
		respBytes = SerializeInteger(respBytes, 1)
	} else {
		// negative request id: expecting no response
		respBytes = nil
	}

	// handle service
	respBytes = srv.handleService(ctx, requestId, serviceId, functionId, requestBytes, respBytes)

	// if request id <= 0, always return nil
	if requestId <= 0 {
		return nil
	}

	// if request failed but client expects a response, return a failed result instead of nil
	if respBytes == nil {
		respBytes = SerializeInteger(originalResp, requestId)
		respBytes = SerializeInteger(respBytes, 0)
	}

	// done
	return respBytes
}
