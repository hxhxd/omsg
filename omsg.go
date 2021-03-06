package omsg

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

// ServerInterface 服务端接口
type ServerInterface interface {
	OmsgNewClient(conn net.Conn)                          // 新客户端回调
	OmsgData(conn net.Conn, cmd, ext uint16, data []byte) // 数据回调
	OmsgError(conn net.Conn, err error)                   // 错误回调
	OmsgClientClose(conn net.Conn)                        // 客户端断开回调
}

// ClientInterface 客户端接口
type ClientInterface interface {
	OmsgData(cmd, ext uint16, data []byte) // 收到命令行回调
	OmsgError(err error)                   // 错误回调
	OmsgClose()                            // 连接断开回调
}

type head struct {
	Sign uint16 // 2数据标志 HK
	CRC  uint16 // 2简单crc校验值
	Cmd  uint16 // 2指令代码
	Ext  uint16 // 2自定义扩展
	Size uint32 // 4数据大小
}

const signWord = 0x4B48            // 标志HK
var headSize = binary.Size(head{}) // 头尺寸

// Send ...
func Send(conn net.Conn, cmd, ext uint16, data []byte) error {
	buffer := make([]byte, headSize+len(data))
	// defer func() { log.Println("send:\n" + hex.Dump(buffer)) }()

	// Sign
	binary.LittleEndian.PutUint16(buffer, signWord)

	// CRC
	binary.LittleEndian.PutUint16(buffer[2:], crc(data))

	// Cmd
	binary.LittleEndian.PutUint16(buffer[4:], cmd)

	// Ext
	binary.LittleEndian.PutUint16(buffer[6:], ext)

	// data length
	binary.LittleEndian.PutUint32(buffer[8:], uint32(len(data)))

	// data
	copy(buffer[headSize:], data)

	// send
	if _, err := conn.Write(buffer); err != nil {
		return err
	}

	return nil
}

// Recv ...
func Recv(conn net.Conn) (uint16, uint16, []byte, error) {

	header := make([]byte, headSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}
	// log.Println("recv header:\n" + hex.Dump(header))

	// Sign
	if signWord != binary.LittleEndian.Uint16(header) {
		return 0, 0, nil, errors.New("sign err")
	}

	// CRC
	icrc := binary.LittleEndian.Uint16(header[2:])

	// Cmd
	cmd := binary.LittleEndian.Uint16(header[4:])

	// Ext
	ext := binary.LittleEndian.Uint16(header[6:])

	// data length
	size := binary.LittleEndian.Uint32(header[8:])

	// data
	buffer := make([]byte, int(size))
	if _, err := io.ReadFull(conn, buffer); err != nil {
		return 0, 0, nil, err
	}
	// log.Println("recv buffer:\n" + hex.Dump(buffer))

	// check crc
	if icrc != crc(buffer) {
		return 0, 0, nil, errors.New("crc err")
	}

	return cmd, ext, buffer, nil
}

func crc(data []byte) uint16 {
	size := len(data)
	crc := uint16(0xFFFF)
	for i := 0; i < size; i++ {
		crc = (crc >> 8) ^ uint16(data[i])
		for j := 0; j < 8; j++ {
			flag := crc & 0x0001
			crc >>= 1
			if flag == 1 {
				crc ^= 0xA001
			}
		}
	}
	return crc
}
