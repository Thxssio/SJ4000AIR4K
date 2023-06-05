package libipcamera

import (
	"fmt"
	"net"
)

func ExampleCreateCamera() {
	cameraIP := net.ParseIP("192.168.0.1")
	camera := CreateCamera(cameraIP, 6666, "admin", "12345")
	defer camera.Disconnect()
	camera.SetVerbose(true)
	camera.Connect()
	err := camera.Login()
	if err != nil {
		fmt.Printf("Falha ao fazer login: %s\n", err)
	}

	err = camera.TakePicture()
	if err != nil {
		fmt.Printf("Falha ao tirar uma foto: %s\n", err)
	}
}

func ExampleCreatePacket() {

	header := CreateCommandHeader(TAKE_PICTURE)
	payload := []byte{}
	packet := CreatePacket(header, payload)
	fmt.Printf("Packet Data: %X\n", packet)
}
