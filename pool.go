package connPool

import "net"

//定义pool接口
type Pool interface {
	//获取
	Get() (net.Conn, error)
	//放回
	put(net.Conn) error
	//关闭
	Close()
	//重置
	Reset(int, int, Factory) error
}
