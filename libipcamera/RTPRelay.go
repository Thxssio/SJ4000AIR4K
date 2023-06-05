package libipcamera

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"net"
	"time"
)

type RTPRelay struct {
	close      bool
	targetIP   net.IP
	targetPort int
	listener   net.PacketConn
	context    context.Context
}

var close bool


func CreateRTPRelay(ctx context.Context, targetAddress net.IP, targetPort int) *RTPRelay {
	conn, err := net.ListenPacket("udp", ":6669")

	if err != nil {
		log.Printf("ERRO: %s\n", err)
	}
	
	close = false
	relay := RTPRelay{
		close: false,
		targetIP:   targetAddress,
		targetPort: targetPort,
		listener:   conn,
		context:    ctx,
	}
	if err != nil {
		log.Printf("ERRO: %s\n", err)
	}

	go handleCameraStream(relay, conn)

	return &relay
}

func handleCameraStream(relay RTPRelay, conn net.PacketConn) {
	buffer := make([]byte, 2048)
	packetReader := bytes.NewReader(buffer)

	header := streamHeader{}
	var payload []byte

	rtpTarget := net.UDPAddr{
		IP:   relay.targetIP,
		Port: relay.targetPort,
	}
	rtpSource, _ := net.ResolveUDPAddr("udp", "127.0.0.1")
	rtpConn, err := net.DialUDP("udp", rtpSource, &rtpTarget)
	if err != nil {
		log.Printf("ERRO ao criar remetente RTP: %s\n", err)
	}

	var sequenceNumber uint16
	var elapsed uint32

	frameBuffer := bytes.Buffer{}
	packetBuffer := bytes.Buffer{}
	T:
		for {
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			
			select {
			case <-relay.context.Done():
				log.Println("Contexto Feito")
				rtpConn.Close()
				relay.listener.Close()
				break T
			default:
				if close {
					rtpConn.Close()
					relay.listener.Close()
					break T
				}
				
				conn.ReadFrom(buffer)
				packetReader.Reset(buffer)

				binary.Read(packetReader, binary.BigEndian, &header)

				if header.Magic != 0xBCDE {
					log.Printf("Mensagem recebida como inválida (%x).", header.Magic)
					break
				}

				if header.Length > 0 {
					payload = make([]byte, header.Length)
					_, err := io.ReadFull(packetReader, payload)
					if err != nil {
						log.Printf("Erro de leitura: %s\n", err)
						break
					}
				} else {
					payload = []byte{}
				}

				switch header.MessageType {
				case 0x0001: 
					frameBuffer.Write(payload)
				case 0x0002: 
					
					packetBuffer.Write(frameBuffer.Bytes())
					rtpConn.Write(packetBuffer.Bytes())
					packetBuffer.Reset()
					packetBuffer.Write([]byte{0x80, 0x63})
					binary.Write(&packetBuffer, binary.BigEndian, sequenceNumber+1)
					binary.Write(&packetBuffer, binary.BigEndian, (uint32)(elapsed)*90)
					binary.Write(&packetBuffer, binary.BigEndian, (uint64(0)))
					frameBuffer.Reset()
					sequenceNumber++

					elapsed = binary.LittleEndian.Uint32(payload[12:])
				default:
					log.Printf("Mensagem desconhecida recebida: %+v\n", header)
					log.Printf("Carga útil:\n%s\n", hex.Dump(payload))
				}
			}
		}
}


func (r *RTPRelay) Stop() {
	close = true
	r.close = true
	r.listener.Close()
}
