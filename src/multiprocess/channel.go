package multiprocess

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

const (
	maxFrameSize = 16 << 20
	writeTimeout = 5 * time.Second
)

// Channel transports framed control and statistics messages over a Unix
// socketpair. Send is safe for concurrent heartbeat and stats publishers.
type Channel struct {
	conn net.Conn
	mu   sync.Mutex
}

func newChannel(conn net.Conn) *Channel {
	return &Channel{conn: conn}
}

func (c *Channel) Close() error {
	return c.conn.Close()
}

func (c *Channel) Send(message Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) > maxFrameSize {
		return errors.New("IPC message exceeds maximum frame size")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	defer c.conn.SetWriteDeadline(time.Time{})

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if err := writeAll(c.conn, header); err != nil {
		return err
	}
	return writeAll(c.conn, payload)
}

func (c *Channel) Receive() (Message, error) {
	var message Message
	header := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return message, err
	}
	size := binary.BigEndian.Uint32(header)
	if size == 0 || size > maxFrameSize {
		return message, errors.New("invalid IPC frame size")
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return message, err
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return message, err
	}
	return message, nil
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrUnexpectedEOF
		}
		data = data[written:]
	}
	return nil
}
