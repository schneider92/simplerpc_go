package simplerpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testService struct {
	id       int64
	revision string

	value     string
	addresult int64
}

const id_testfunc_append_string = 1
const id_testfunc_add_nums = 2
const id_testfunc_wait_a_little = 3

func (srv *testService) GetServiceId() int64 {
	return srv.id
}
func (srv *testService) GetRevision() string {
	return srv.revision
}
func (srv *testService) CallFunction(ctx context.Context, functionId int64, requestBytes []byte, respBytes []byte) []byte {
	// append string
	if functionId == id_testfunc_append_string {
		_, str := DeserializeString(requestBytes)
		srv.value += str
		return SerializeString(respBytes, srv.value)
	}

	// add nums
	if functionId == id_testfunc_add_nums {
		requestBytes, v1 := DeserializeInteger(requestBytes)
		_, v2 := DeserializeInteger(requestBytes)
		srv.addresult = v1 + v2
		return SerializeInteger(respBytes, srv.addresult)
	}

	// wait a little
	if functionId == id_testfunc_wait_a_little {
		time.Sleep(time.Millisecond * 200)
		return respBytes
	}

	// no such func
	return nil
}

func TestGetServicesWhenEmpty(t *testing.T) {
	// get services when there are no services

	// create server
	server, err := NewServer([]ServerService{})
	assert.Nil(t, err)

	// get services
	req := []byte{
		0x16, // request id
		0,    // service id
		0,    // function id
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{})

	// validate result
	assert.Equal(t, []byte{
		0x16, // request id
		1,    // success
		0,    // 0 services
	}, resp)
}

func TestGetServices(t *testing.T) {
	// get services when there are 2 services

	// create services
	srv1 := &testService{
		id:       6,
		revision: "rev",
	}
	srv2 := &testService{
		id:       9,
		revision: "asdf",
	}

	// create server
	server, err := NewServer([]ServerService{srv1, srv2})
	assert.Nil(t, err)

	// get services
	req := []byte{
		0xc, // request id
		0,   // service id
		0,   // function id
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{})

	// validate result
	assert.Equal(t, []byte{
		0xc,              // request id
		1,                // success
		2,                // 2 services
		6,                // first service, id
		3, 'r', 'e', 'v', // first service, revision
		9,                     // seconds service, id
		4, 'a', 's', 'd', 'f', // second service, revision
	}, resp)
}

func TestBadServiceId(t *testing.T) {
	// cannot register a service with id=0
	srv := &testService{
		id:       0,
		revision: "myrev",
	}
	_, err := NewServer([]ServerService{srv})
	assert.NotNil(t, err)

	// cannot register a service with id<0
	srv.id = -2
	_, err = NewServer([]ServerService{srv})
	assert.NotNil(t, err)
}

func TestServiceIdCollision(t *testing.T) {
	// cannot register two services with the same id
	srv1 := &testService{
		id:       666,
		revision: "asdf",
	}
	srv2 := &testService{
		id:       666,
		revision: "qwer",
	}
	_, err := NewServer([]ServerService{srv1, srv2})
	assert.NotNil(t, err)

	// different id, same revision is ok
	srv2.id = 777
	srv2.revision = srv1.revision
	_, err = NewServer([]ServerService{srv1, srv2})
	assert.Nil(t, err)
}

func TestServerEcho(t *testing.T) {
	// test the server echo test function

	// create server
	server, err := NewServer([]ServerService{})
	assert.Nil(t, err)

	// echo
	req := []byte{
		1,         // request id
		0,         // service id
		2,         // function id: echo
		0x20, 100, // first argument: wait time = 100ms
		4, 'a', 's', 'd', 'f', // second argument: string to echo
	}

	// run and measure time
	t0 := time.Now()
	resp := server.ProcessRequest(context.Background(), req, []byte{})
	t1 := time.Now()
	diff := t1.Sub(t0).Milliseconds()
	assert.GreaterOrEqual(t, int(diff), 100)

	// validate result
	assert.Equal(t, []byte{
		1,                     // request id
		1,                     // success
		4, 'a', 's', 'd', 'f', // return value
	}, resp)
}

func TestCallFunctionOnService(t *testing.T) {
	// create services
	srv1 := &testService{
		id:       1,
		revision: "1",
		value:    "asdf",
	}
	srv2 := &testService{
		id:       2,
		revision: "2",
		value:    "qwer",
	}

	// create server
	server, err := NewServer([]ServerService{srv1, srv2})
	assert.Nil(t, err)

	//
	//
	// add 2 numbers with the first service
	req := []byte{
		1,                    // request id
		1,                    // service id
		id_testfunc_add_nums, // function id
		6,                    // num 1=6
		7,                    // num 2=7
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{6, 6, 6})
	assert.Equal(t, []byte{
		6, 6, 6, // prefix in the response buffer
		1,  // request id
		1,  // success
		13, // result
	}, resp)

	//
	//
	// add 2 numbers with the second service
	req = []byte{
		1,                    // request id
		2,                    // service id
		id_testfunc_add_nums, // function id
		0x20, 90,             // num 1=90
		20, // num 2=20
	}
	resp = server.ProcessRequest(context.Background(), req, []byte{})
	assert.Equal(t, []byte{
		1,         // request id
		1,         // success
		0x20, 110, // result=110
	}, resp)

	//
	//
	// append string with the first service
	req = []byte{
		1,                         // request id
		1,                         // service id
		id_testfunc_append_string, // function id
		3, '1', '2', '3',          // string data
	}
	resp = server.ProcessRequest(context.Background(), req, nil)
	assert.Equal(t, []byte{
		1,                                    // request id
		1,                                    // success
		7, 'a', 's', 'd', 'f', '1', '2', '3', // string data
	}, resp)

	//
	//
	// append string with the second service
	req = []byte{
		1,                         // request id
		2,                         // service id
		id_testfunc_append_string, // function id
		3, '1', '2', '3',          // string data
	}
	resp = server.ProcessRequest(context.Background(), req, nil)
	assert.Equal(t, []byte{
		1,                                    // request id
		1,                                    // success
		7, 'q', 'w', 'e', 'r', '1', '2', '3', // string data
	}, resp)
}

func TestCallFunctionOnServiceNoResponse(t *testing.T) {
	// create server
	service := &testService{
		id:       1,
		revision: "1",
		value:    "asdf",
	}
	server, _ := NewServer([]ServerService{service})

	// add 2 numbers
	req := []byte{
		0,                    // request id 0, expect no response
		1,                    // service id
		id_testfunc_add_nums, // function id
		11,                   // num 1=11
		4,                    // num 2=4
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{})
	assert.Nil(t, resp)

	// check the result
	assert.Equal(t, 15, int(service.addresult))

	// append string
	req = []byte{
		0,                         // request id 0
		1,                         // service id
		id_testfunc_append_string, // function id
		2, 'g', 'h',               // data
	}
	resp = server.ProcessRequest(context.Background(), req, nil)
	assert.Nil(t, resp)

	// check the result
	assert.Equal(t, "asdfgh", service.value)
}

func TestWaitALittle(t *testing.T) {
	// test the test function

	// create server
	server, _ := NewServer([]ServerService{
		&testService{
			id: 1,
		},
	})

	// request
	req := []byte{
		1,                         // request id
		1,                         // service id
		id_testfunc_wait_a_little, // func id
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{})

	// check response
	assert.Equal(t, []byte{
		1, // request id
		1, // valid value (but no real value as the function did not return anything, but it succeeded)
	}, resp)
}

func TestCancel(t *testing.T) {
	// create server
	server, _ := NewServer([]ServerService{
		&testService{
			id: 1,
		},
	})

	// call wait a little function on a different goroutine
	done := make(chan []byte)
	go func() {
		req := []byte{
			1,                         // request id
			1,                         // service id
			id_testfunc_wait_a_little, // func id
		}
		resp := server.ProcessRequest(context.Background(), req, []byte{})

		// finished
		done <- resp
	}()

	// wait 100ms to make sure it started
	time.Sleep(time.Millisecond * 100)

	// send cancel
	req := []byte{
		0, // request id, zero as we don't expect response to a cancel request
		0, // service id
		1, // func id (=cancel)
		1, // request id to cancel
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{})

	// cancel request should return nil, even if a buffer was provided to it
	assert.Nil(t, resp)

	// wait for the goroutine to finish and get its response
	resp = <-done

	// check response
	assert.Equal(t, []byte{
		1, // request id
		0, // invalid value
	}, resp)
}

func TestInvalidRequest(t *testing.T) {
	// create server
	server, _ := NewServer([]ServerService{})

	// request
	req := []byte{
		1, // request id
		0, // service id
		// and the missing fields ...
	}
	resp := server.ProcessRequest(context.Background(), req, []byte{})
	assert.Nil(t, resp)
}

func TestInvalidId(t *testing.T) {
	// create server
	server, _ := NewServer([]ServerService{
		&testService{
			id: 1,
		},
	})

	// request function 4 on service 0 (the server service)
	req := []byte{
		1, // request id
		0, // service id
		4, // function id
	}
	resp := server.ProcessRequest(context.Background(), req, nil)
	assert.Equal(t, []byte{
		1, // request id
		0, // failed
	}, resp)

	// request function 4 on service 1 (test service, this tests the test actually)
	req = []byte{
		1, // request id
		1, // service id
		4, // function id
	}
	resp = server.ProcessRequest(context.Background(), req, nil)
	assert.Equal(t, []byte{
		1, // request id
		0, // failed
	}, resp)

	// request any function on service 2 (no such service)
	req = []byte{
		1, // request id
		2, // service id
		1, // function id
	}
	resp = server.ProcessRequest(context.Background(), req, nil)
	assert.Equal(t, []byte{
		1, // request id
		0, // failed
	}, resp)
}
