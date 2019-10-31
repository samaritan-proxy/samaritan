package tcp

import "net"

func newLocalListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

type localServer struct {
	net.Listener
	done chan bool
}

func newLocalServer() (*localServer, error) {
	ln, err := newLocalListener()
	if err != nil {
		return nil, err
	}
	ls := &localServer{
		Listener: ln,
		done:     make(chan bool),
	}
	return ls, nil
}

func (ls *localServer) tearDown() {
	if ls.Listener != nil {
		ls.Listener.Close()
		<-ls.done
		ls.Listener = nil
	}
}

func (ls *localServer) buildup(handler func(*localServer)) {
	go func() {
		handler(ls)
		close(ls.done)
	}()
}

func (ls *localServer) Network() string {
	return ls.Listener.Addr().Network()
}

func (ls *localServer) Address() string {
	return ls.Listener.Addr().String()
}
