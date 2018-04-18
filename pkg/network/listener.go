package network

import (
	"context"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"net"
	"runtime/debug"
<<<<<<< HEAD
	"time"
=======
>>>>>>> origin/master
)

// listener impl based on golang net package
type listener struct {
	name                                  string
	localAddress                          net.Addr
	bindToPort                            bool
	listenerTag                           uint64
	perConnBufferLimitBytes               uint32
	handOffRestoredDestinationConnections bool
	cb                                    types.ListenerEventListener
	rawl                                  *net.TCPListener
}

func NewListener(lc *v2.ListenerConfig) types.Listener {

	l := &listener{
		name:                                  lc.Name,
		localAddress:                          lc.Addr,
		bindToPort:                            lc.BindToPort,
		listenerTag:                           lc.ListenerTag,
		perConnBufferLimitBytes:               lc.PerConnBufferLimitBytes,
		handOffRestoredDestinationConnections: lc.HandOffRestoredDestinationConnections,
	}

	if lc.InheritListener != nil {
		//inherit old process's listener
		l.rawl = lc.InheritListener
	}
	return l
}

/*
func NewFDListener(lc *v2.ListenerConfig, fd uintptr) types.Listener {
	s := &Server{cm: NewConnectionManager(), logger: logger}

	file := os.NewFile(fd, "/tmp/sock-go-graceful-restart")
	listener, err := net.FileListener(file)
	if err != nil {
		return nil, errors.New("File to recover socket from file descriptor: " + err.Error())
	}
	listenerTCP, ok := listener.(*net.TCPListener)
	if !ok {
		return nil, fmt.Errorf("File descriptor %d is not a valid TCP socket", fd)
	}
	s.socket = listenerTCP

	return s, nil
}
*/

func (l *listener) Name() string {
	return l.name
}

func (l *listener) Addr() net.Addr {
	return l.localAddress
}

func (l *listener) Start(stopChan chan bool, lctx context.Context) {
	//call listen if not inherit
	if l.rawl == nil {
		if err := l.listen(lctx); err != nil {
			// TODO: notify listener callbacks
			log.DefaultLogger.Println(l.name+" listen failed, ", err)
			return
		}
	}

	if l.bindToPort {
		for {
<<<<<<< HEAD
			//select {
			//case <-stopChan:
			//	//FIXME: can not enter this branch util Listener.accept return
			//	log.DefaultLogger.Println("listener " +l.name + " stop accepting connections by stop chan")
			//	return
			//default:
			if err := l.accept(lctx); err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					log.DefaultLogger.Println("listener " + l.name + " stop accepting connections by deadline")
					return
				} else if ope, ok := err.(*net.OpError); ok {
					if !(ope.Timeout() && ope.Temporary()) {
						log.DefaultLogger.Println("not temp-timeout error:" + err.Error())
=======
			select {
			case <-stopChan:
				log.DefaultLogger.Println("listener " + l.name + " stop accepting connections")
				return
			default:
				if err := l.accept(lctx); err != nil {
					if ope, ok := err.(*net.OpError); ok {
						if !(ope.Timeout() && ope.Temporary()) {
							log.DefaultLogger.Println("not temp-timeout error:" + err.Error())
						}
					} else {
						log.DefaultLogger.Println("unknown error while listener accepting:" + err.Error())
>>>>>>> origin/master
					}
				} else {
					log.DefaultLogger.Println("unknown error while listener accepting:" + err.Error())
				}
			}
			//}
		}
	}
}

func (l *listener) Stop() {
	l.rawl.SetDeadline(time.Now())
}

func (l *listener) ListenerTag() uint64 {
	return l.listenerTag
}

func (l *listener) ListenerFD() (uintptr, error) {
	file, err := l.rawl.File()
	if err != nil {
		log.DefaultLogger.Println(l.name, " listener fd not found : ", err)
		return 0, err
	}
	return file.Fd(), nil
}

func (l *listener) PerConnBufferLimitBytes() uint32 {
	return l.perConnBufferLimitBytes
}

func (l *listener) SetListenerCallbacks(cb types.ListenerEventListener) {
	l.cb = cb
}

func (l *listener) Close(lctx context.Context) error {
	l.cb.OnClose()
	return l.rawl.Close()
}

func (l *listener) listen(lctx context.Context) error {
	var err error

	var rawl *net.TCPListener
	if rawl, err = net.ListenTCP("tcp", l.localAddress.(*net.TCPAddr)); err != nil {
		return err
	}

	l.rawl = rawl

	return nil
}

func (l *listener) accept(lctx context.Context) error {
	rawc, err := l.rawl.Accept()

	if err != nil {
		return err
	}

	// TODO: use thread pool
	go func() {
		defer func() {
			if p := recover(); p != nil {
				fmt.Printf("panic %v", p)
				fmt.Println()

				debug.PrintStack()
			}
		}()

		l.cb.OnAccept(rawc, l.handOffRestoredDestinationConnections)
	}()

	return nil
}
