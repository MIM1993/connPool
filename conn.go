package connPool

import (
	"net"
	"sync"
)

//从pool中取出的conn的包装器
type PoolConn struct {
	net.Conn               //继承了这个接口
	rl       *sync.RWMutex //读写锁，用来在改变链接状态时使用
	p        *chanPool     //pool
	unusable bool          //是否将链接关闭
}

//将链接放回pool中或者直接关闭
func (p *PoolConn) Closs() error {
	//加读锁
	p.rl.RLock()
	defer p.rl.RUnlock()
	var err error
	if p.unusable {
		if p.Conn != nil { //有必要吗？
			err = p.Conn.Close()
		}
		return err
	}
	return p.p.put(p.Conn)
}

//修改链接状态
func (p *PoolConn) MarkUnusable() {
	//加写锁,修改时不让其他线程读写
	p.rl.Lock()
	p.unusable = true
	p.rl.Unlock()
}

//包装器
func (c *chanPool) WarpConn(conn net.Conn) net.Conn {
	p := &PoolConn{}
	p.rl = &sync.RWMutex{}
	p.p = c
	p.Conn = conn
	return p
}