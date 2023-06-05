package libipcamera

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/icza/bitio"
)


type Header struct {
	Magic       uint16
	Length      uint16
	MessageType uint32
}

func (h *Header) String() string {
	return fmt.Sprintf("{ Header Magic=0x%X, Length=%d, MessageType=0x%X }", h.Magic, h.Length, h.MessageType)
}


type Message struct {
	Header  Header
	Payload []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("{ Message\n\tHeader=%s,\n\tPayload=\n%s\n}", m.Header.String(), hex.Dump(m.Payload))
}


type streamHeader struct {
	Magic          uint16
	Length         uint16
	SequenceNumber uint16
	MessageType    uint16
}


func CreatePacket(header Header, payload []byte) []byte {
	header.Length = (uint16)(len(payload))

	buf := &bytes.Buffer{}
	w := bitio.NewWriter(buf)
	w.WriteBits((uint64)(header.Magic), 16)
	w.WriteBits((uint64)(header.Length), 16)
	w.WriteBits((uint64)(header.MessageType), 32)
	w.Write(payload)
	return buf.Bytes()
}


func CreateCommandHeader(command uint32) Header {
	return Header{
		Magic:       0xABCD,
		Length:      0,
		MessageType: command,
	}
}


func CreateLoginPacket(username, password string) []byte {
	header := CreateCommandHeader(LOGIN) // Login
	payload := make([]byte, 128)
	copy(payload, []byte(username))
	copy(payload[64:], []byte(password))

	return CreatePacket(header, payload)
}


func CreateCommandPacket(command uint32) []byte {
	header := CreateCommandHeader(command)
	return CreatePacket(header, []byte{})
}
