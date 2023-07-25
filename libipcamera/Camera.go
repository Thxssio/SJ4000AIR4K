package libipcamera

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)


type Camera struct {
	ipAddress       net.IP
	port            int
	username        string
	password        string
	connected       bool
	disconnect      bool
	verbose         bool
	connection      net.Conn
	isLoggedIn      bool
	messageHandlers map[uint32][]MessageHandler
}


type MessageHandler func(camera *Camera, message *Message) (bool, error)

const (
	LOGIN                 = 0x0110
	LOGIN_ACCEPT          = 0x0111
	ALIVE_REQUEST         = 0x0112
	ALIVE_RESPONSE        = 0x0113
	DISCOVERY_REQUEST     = 0x0114
	DISCOVERY_RESPONSE    = 0x0115
	START_PREVIEW         = 0x01FF
	REQUEST_FILE_LIST     = 0xA025
	FILE_LIST_CONTENT     = 0xA026
	REQUEST_FIRMWARE_INFO = 0xA034
	FIRMWARE_INFORMATION  = 0xA035
	TAKE_PICTURE          = 0xA038
	PICTURE_SAVED         = 0xA039
	CONTROL_RECORDING     = 0xA03A
	RECORD_COMMAND_ACCEPT = 0xA03B
)

const (

	RemoveHandler = true

	KeepHandler = false
)


type StoredFile struct {
	Path string
	Size uint64
}

func CreateCamera(ipAddress net.IP, port int, username, password string) (*Camera, error) {
	if ipAddress == nil {
		return nil, errors.New("Não é possível criar uma câmera sem um endereço IP")
	}
	camera := &Camera{
		ipAddress:       ipAddress,
		port:            port,
		username:        username,
		password:        password,
		messageHandlers: make(map[uint32][]MessageHandler, 0),
		verbose:         true,
	}
	return camera, nil
}


func (c *Camera) Connect() {
	if c.verbose {
		log.Printf("Conectando à %s:%d usando nome de usuário =%s, senha=%s\n", c.ipAddress, c.port, c.username, c.password)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.ipAddress, c.port))
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return
	}
	c.connection = conn

	c.HandleFirst(ALIVE_REQUEST, aliveRequestHandler)

	go c.handleConnection()
}


func (c *Camera) Login() error {
	loginAccept := make(chan bool, 0)

	c.Handle(LOGIN_ACCEPT, func(c *Camera, m *Message) (bool, error) {
		_, err := loginResultHandler(c, m)
		if err != nil {
			return RemoveHandler, err
		}
		loginAccept <- true
		return RemoveHandler, nil
	})

	c.SendPacket(CreateLoginPacket(c.username, c.password))

	select {
	case <-loginAccept:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("A solicitação de login expirou")
	}
}

func (c *Camera) IsConnected() bool {
	return c.connected
}

func (c *Camera) handleConnection() {
	header := Header{}
	var payload []byte

	for {
		if c.disconnect {
			break
		}


		err := binary.Read(c.connection, binary.BigEndian, &header)
		if err != nil {
			if !c.disconnect {
				log.Printf("ERRO ao ler da câmera: %s\n", err)
			}
			break
		}

	
		if header.Magic != 0xABCD {
			log.Printf("Mensagem recebida como inválida (%x)\n", header.Magic)
			break
		}


		if header.Length > 0 {
			payload = make([]byte, header.Length)
			bytesRead, err := io.ReadFull(c.connection, payload)
			if err != nil || (uint16(bytesRead) != header.Length) {
				log.Printf("ERRO ao ler a carga útil da câmera: %s, expected %d Bytes, got %d\n", err, header.Length, bytesRead)
				break
			}
		} else {
			payload = []byte{}
		}

		message := &Message{
			Header:  header,
			Payload: payload,
		}

		if len(c.messageHandlers[header.MessageType]) == 0 {
			log.Printf("Mensagem desconhecida recebida (nenhum manipulador registrado):\n%s\n", message)
			continue
		}


		remainingMessageHandlers := make([]MessageHandler, 0)
		for _, handler := range c.messageHandlers[header.MessageType] {
			remove, err := handler(c, message)
			if remove == KeepHandler {
				remainingMessageHandlers = append(remainingMessageHandlers, handler)
			}

			if err != nil {
				log.Printf("ERRO ao executar o manipulador de mensagens (%v): %s\n", handler, err)
				break
			}
		}
	
		c.messageHandlers[header.MessageType] = remainingMessageHandlers
	}
	c.Log("Desconectado")
	c.connected = false
}


func (c *Camera) Handle(messageType uint32, handleFunc MessageHandler) {
	c.addHandler(messageType, handleFunc, false)
}


func (c *Camera) HandleFirst(messageType uint32, handleFunc MessageHandler) {
	c.addHandler(messageType, handleFunc, true)
}

func (c *Camera) addHandler(messageType uint32, handleFunc MessageHandler, prepend bool) {
	if c.messageHandlers[messageType] == nil {
		c.messageHandlers[messageType] = make([]MessageHandler, 0)
	}

	if prepend {
		c.messageHandlers[messageType] = append([]MessageHandler{handleFunc}, c.messageHandlers[messageType]...)
	} else {
		c.messageHandlers[messageType] = append(c.messageHandlers[messageType], handleFunc)
	}
}


func (c *Camera) Log(format string, data ...interface{}) {
	if c.verbose {
		if data != nil {
			log.Printf(format+"\n", data)
		} else {
			log.Printf(format + "\n")
		}
	}
}


func (c *Camera) GetFileList() ([]StoredFile, error) {
	fileListComplete := make(chan []StoredFile, 1)
	fileListData := ""

	c.Handle(FILE_LIST_CONTENT, func(c *Camera, m *Message) (bool, error) {
		numParts := binary.LittleEndian.Uint32(m.Payload[:4])
		currentPart := binary.LittleEndian.Uint32(m.Payload[4:8])
		fileListData += string(m.Payload[8:])
		if currentPart+1 >= numParts {
			fileListComplete <- parseFileList(fileListData)
			return RemoveHandler, nil
		}
		return KeepHandler, nil
	})

	err := c.SendPacket(CreatePacket(CreateCommandHeader(REQUEST_FILE_LIST), []byte{0x01, 0x00, 0x00, 0x00}))
	if err != nil {
		return nil, err
	}

	select {
	case result := <-fileListComplete:
		return result, nil
	case <-time.After(10 * time.Second):
		return nil, errors.New("Tempo esgotado ao carregar a lista de arquivos")
	}
}

func parseFileList(input string) []StoredFile {
	files := strings.Split(input, ";")
	stored := make([]StoredFile, len(files)-1)
	for i, file := range files {
		parts := strings.Split(file, ":")
		if len(parts) == 2 {
			size, err := strconv.ParseUint(parts[1], 10, 64)

			if err == nil && size > 0 && len(parts[0]) > 0 {
				stored[i] = StoredFile{
					Path: parts[0],
					Size: size,
				}
			}
		}
	}
	return stored
}


func (c *Camera) GetFirmwareInfo() (string, error) {
	if !c.isLoggedIn {
		return "", errors.New("É necessário fazer login na câmera")
	}

	firmwareInfo := make(chan string, 1)
	c.Handle(FIRMWARE_INFORMATION, func(c *Camera, m *Message) (bool, error) {
		firmwareInfo <- string(m.Payload)
		return RemoveHandler, nil
	})
	err := c.SendPacket(CreateCommandPacket(REQUEST_FIRMWARE_INFO))
	if err != nil {
		return "", err
	}

	select {
	case result := <-firmwareInfo:
		return result, nil
	case <-time.After(5 * time.Second):
		return "", errors.New("Solicitação de informações de firmware expirou")
	}
}


func (c *Camera) SendPacket(packet []byte) error {
	_, err := c.connection.Write(packet)
	return err
}


func (c *Camera) TakePicture() error {
	if !c.isLoggedIn {
		return errors.New("É necessário fazer login na câmera")
	}

	pictureTaken := make(chan bool, 1)
	c.Handle(PICTURE_SAVED, func(c *Camera, m *Message) (bool, error) {
		c.Log("A imagem foi salva no cartão SD")
		pictureTaken <- true
		return RemoveHandler, nil
	})

	err := c.SendPacket(CreateCommandPacket(TAKE_PICTURE))
	if err != nil {
		return err
	}

	select {
	case <-pictureTaken:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("A solicitação TAKE_PICTURE expirou")
	}
}

func (c *Camera) StartPreviewStream() error {
	if !c.isLoggedIn {
		return errors.New("É necessário fazer login na câmera")
	}
	c.Log("Iniciando fluxo de visualização")
	return c.SendPacket(CreateCommandPacket(START_PREVIEW))
}


func (c *Camera) StartRecording() error {
	if !c.isLoggedIn {
		return errors.New("É necessário fazer login na câmera")
	}

	recordCommandAccept := make(chan bool, 1)

	c.Handle(RECORD_COMMAND_ACCEPT, func(c *Camera, m *Message) (bool, error) {
		c.Log("Começou a gravar vídeo")
		recordCommandAccept <- true
		return RemoveHandler, nil
	})

	c.Log("Solicitando câmera para iniciar a gravação")
	err := c.SendPacket(CreatePacket(CreateCommandHeader(CONTROL_RECORDING), []byte{0x01, 0x00, 0x00, 0x00}))
	if err != nil {
		return err
	}

	select {
	case <-recordCommandAccept:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("A solicitação de CONTROL_RECORDING expirou")
	}
}

// StopRecording stops recording video to SD-Card
func (c *Camera) StopRecording() error {
	if !c.isLoggedIn {
		return errors.New("É necessário fazer login na câmera")
	}

	recordCommandAccept := make(chan bool, 1)

	c.Handle(RECORD_COMMAND_ACCEPT, func(c *Camera, m *Message) (bool, error) {
		c.Log("Parando de gravar vídeo")
		recordCommandAccept <- true
		return RemoveHandler, nil
	})

	c.Log("Solicitando que a câmera pare de gravar")
	err := c.SendPacket(CreatePacket(CreateCommandHeader(CONTROL_RECORDING), []byte{0x00, 0x00, 0x00, 0x00}))
	if err != nil {
		return err
	}

	select {
	case <-recordCommandAccept:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("A solicitação de CONTROL_RECORDING expirou")
	}
}


func (c *Camera) Disconnect() {
	c.disconnect = true
	c.connected = false
	c.connection.Close()
}


func (c *Camera) SetVerbose(verbose bool) {
	c.verbose = verbose
}

func aliveRequestHandler(camera *Camera, message *Message) (bool, error) {
	responseHeader := CreateCommandHeader(ALIVE_RESPONSE)
	response := CreatePacket(responseHeader, []byte{})
	return KeepHandler, camera.SendPacket(response)
}


func firmwareInfoHandler(camera *Camera, message *Message) (bool, error) {
	camera.Log("Informações de firmware recebidas")
	camera.Log("Firmware Version: %s", string(message.Payload))
	return KeepHandler, nil
}

func loginResultHandler(camera *Camera, message *Message) (bool, error) {
	if message.Header.MessageType == 0x0111 {
		camera.isLoggedIn = true
		camera.Log("Login Aceito")
	} else if message.Header.MessageType == 0x1234 { 
		camera.Log("Login Falhou")
		return RemoveHandler, errors.New("Já existe um cliente conectado à câmera")
	}
	return RemoveHandler, nil
}
