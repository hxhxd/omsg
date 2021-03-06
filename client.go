package omsg

import (
	"net"
	"time"
)

// Client ...
type Client struct {
	ci   ClientInterface
	Conn net.Conn // 用户客户端
}

// NewClient 创建客户端
func NewClient(ci ClientInterface) *Client {
	return &Client{ci: ci}
}

// Connect 连接到服务器
func (o *Client) Connect(address string) error {
	return o.ConnectTimeout(address, 0)
}

// ConnectTimeout 连接到服务器
func (o *Client) ConnectTimeout(address string, timeout time.Duration) error {
	var err error
	if o.Conn, err = net.DialTimeout("tcp", address, timeout); err != nil {
		return err
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	defer func() {
		o.Close()

		// 回调
		o.ci.OmsgClose()
	}()

	for {
		cmd, ext, bs, err := Recv(o.Conn)
		if err != nil {
			o.ci.OmsgError(err)
			break
		}
		o.ci.OmsgData(cmd, ext, bs)
	}
}

// Close 关闭链接
func (o *Client) Close() {
	o.Conn.Close()
}
