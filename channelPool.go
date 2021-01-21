package connPool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

var (
	// ErrClosed is the error resulting if the pool is closed via pool.Close().
	ErrClosed = errors.New("pool is closed")
)

//具体pool class
type chanPool struct {
	//读写锁
	rl *sync.RWMutex
	//存放链接chan
	conns chan net.Conn
	//生产工厂
	factory Factory
}

func (c *chanPool) Get() (net.Conn, error) {
	conns, factory := c.getFactoryAndConns()

	if conns == nil {
		return nil, errors.New("pool is closed")
	}

	var conn net.Conn
	select {
	case c := <-conns:
		if c == nil {
			return nil, errors.New("pool is closed")
		}
		conn = c
	default:
		if factory != nil {
			c, err := factory()
			if err != nil {
				return nil, fmt.Errorf("pool is nil, create conn err: %s\n", err.Error())
			}
			conn = c
		} else {
			return nil, errors.New("pool factory is nil")
		}
	}
	return c.WarpConn(conn), nil
}

func (c *chanPool) put(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn is nll, return")
	}

	//读锁
	c.rl.RLock()
	defer c.rl.RUnlock()

	if c.conns == nil {
		return conn.Close()
	}

	select {
	case c.conns <- conn:
	default:
		conn.Close()
		return errors.New("pool is full")
	}
	return nil
}

//关闭连接池
func (c *chanPool) Close() {
	c.rl.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.rl.Unlock()

	//关闭channel
	close(conns)
	for v := range conns {
		v.Close()
	}
}

func (c *chanPool) Reset(initSize, maxCap int, factory Factory) error {
	//首先创建pool
	if p, err := newChanPool(initSize, maxCap, factory); err != nil {
		return fmt.Errorf("Reset err:%s", err.Error())
	} else {
		//关闭旧pool
		c.Close()

		//新pool赋值
		c = p.(*chanPool)
	}
	return nil
}

func (c *chanPool) getFactoryAndConns() (chan net.Conn, Factory) {
	c.rl.RLock()
	conns := c.conns
	factory := c.factory
	c.rl.RUnlock()
	return conns, factory
}

//工厂
type Factory func() (net.Conn, error)

func newChanPool(initSize, maxCap int, factory Factory) (Pool, error) {
	if initSize < 0 || maxCap <= 0 || initSize > maxCap || factory == nil {
		return nil, errors.New("Parameter error")
	}
	//初始化chanpool
	cp := &chanPool{
		rl:      &sync.RWMutex{},
		conns:   make(chan net.Conn, maxCap),
		factory: factory,
	}

	for i := 0; i < initSize; i++ {
		conn, err := factory()
		if err != nil {
			cp.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		cp.conns <- conn
	}
	return cp, nil
}

func NewChanPool(initSize, maxCap int, factory Factory) (Pool, error) {
	return newChanPool(initSize, maxCap, factory)
}

//默认参数创建pool
