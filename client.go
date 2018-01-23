package omsg

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"
)

// Client ...
type Client struct {
	client         net.Conn      // 用户客户端
	onData         func([]byte)  // 收到命令行回调
	onClose        func()        // 链接断开回调
	onReconnect    func()        // 重连回调
	serverAddr     string        // 服务器地址
	reconnect      bool          // 断线重连
	connectTimeout time.Duration // 连线超时时间（秒）
	reconnectWait  time.Duration // 重连时间间隔（秒）
	crypt          *crypt
}

// NewClient 创建客户端
func NewClient(key []byte, onData func([]byte), onClose func(), onReconnect func()) *Client {
	return &Client{onData: onData, onClose: onClose, crypt: newCrypt(key)}
}

// Connect 连接到服务器
func (o *Client) Connect(address string, reconnect bool, connectTimeout time.Duration, reconnectWait time.Duration) error {
	o.serverAddr = address
	o.reconnect = reconnect
	o.connectTimeout = connectTimeout
	o.reconnectWait = reconnectWait

	return o.connect(false)
}

func (o *Client) connect(re bool) error {
	for {
		var err error
		if o.client, err = net.DialTimeout("tcp", o.serverAddr, o.connectTimeout); err == nil {
			break
		}
		if !o.reconnect {
			return err
		}
		if re && o.onReconnect != nil {
			o.onReconnect()
		}
		time.Sleep(o.reconnectWait)
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	// 数据缓存
	cache := new(bytes.Buffer)
	buf := make([]byte, 0x100)
	var recvLen int
	var err error
	var needLen int

	for {
		if recvLen, err = o.client.Read(buf); err != nil {
			break
		}

		// 写入缓存
		cache.Write(buf[:recvLen])

		for {
			// 读取数据长度
			if needLen == 0 {
				// 头4字节是数据长度
				if cache.Len() < 8 {
					break
				}

				needLen = int(binary.LittleEndian.Uint32(cache.Next(4))) - 4
			}

			// 数据长度不够，继续读取
			if needLen > cache.Len() {
				break
			}

			// 数据回调
			if o.onData != nil {
				tmp := cache.Next(needLen)
				originLen := int(binary.LittleEndian.Uint32(tmp))
				o.crypt.Decrypt(tmp[4:])
				o.onData(tmp[4 : 4+originLen])
			} else {
				cache.Next(needLen)
			}
			needLen = 0
		}
	}
	o.client.Close()
	if o.reconnect {
		o.connect(true)
	}
	if o.onClose != nil {
		o.onClose()
	}
}

// Send 向服务器发送数据
func (o *Client) Send(data []byte) (int, error) {
	var buf [8]byte
	sizeOld := len(data)
	sizeNew := sizeOld
	if len(data)%16 != 0 {
		sizeNew += 16 - (len(data) % 16)
	}
	// 数据大小
	binary.LittleEndian.PutUint32(buf[:], uint32(sizeNew+0x8))
	// 原数据大小
	binary.LittleEndian.PutUint32(buf[4:], uint32(sizeOld))

	if n, err := o.client.Write(buf[:]); err != nil {
		return n, err
	}

	if sizeNew != sizeOld {
		data = append(data, bytes.Repeat([]byte("0"), sizeNew-sizeOld)...)
	}
	o.crypt.Encrypt(data)

	return o.client.Write(data)
}

// Close 关闭链接
func (o *Client) Close() {
	o.client.Close()
}
