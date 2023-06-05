package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/Thxssio/CamOpen/tree/main/libipcamera"
	"github.com/Thxssio/CamOpen/tree/main/rtsp"
	"github.com/spf13/cobra"
)

func connectAndLogin(ip net.IP, port int, username, password string, verbose bool) *libipcamera.Camera {
	camera, err := libipcamera.CreateCamera(ip, port, username, password)
	if err != nil {
		log.Printf("ERRO ao instanciar câmera: %s\n", err)
		os.Exit(1)
	}
	camera.SetVerbose(verbose)
	camera.Connect()
	camera.Login()

	return camera
}

func main() {
	var username string
	var password string
	var port int16
	var verbose bool
	var cpuprofile string
	var memoryprofile string

	var cpuprofileFile *os.File

	var camera *libipcamera.Camera

	var applicationContext context.Context

	var rootCmd = &cobra.Command{
		Use:   "actioncam [endereço IP das câmeras]",
		Short: "actioncam é uma ferramenta para transmitir a visualização de vídeo de câmeras de ação baratas sem o aplicativo móvel",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			defer camera.Disconnect()
			relay := libipcamera.CreateRTPRelay(applicationContext, net.ParseIP("127.0.0.1"), 5220)
			defer relay.Stop()

			camera.StartPreviewStream()

			bufio.NewReader(os.Stdin).ReadBytes('\n')
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			signalChannel := make(chan os.Signal)
			signal.Notify(signalChannel, os.Interrupt)
			var cancel context.CancelFunc
			applicationContext, cancel = context.WithCancel(context.Background())
			go func(cancel context.CancelFunc) {
				select {
				case sig := <-signalChannel:
					log.Printf("Tem sinal %s, saindo...\n", sig)
					cancel()
					os.Exit(0)
				}
			}(cancel)

			if cpuprofile != "" {
				cpuprofileFile, err := os.Create(cpuprofile)
				if err != nil {
					log.Printf("Não foi possível criar o arquivo de perfil da CPU: %s\n", err)
					return
				}
				err = pprof.StartCPUProfile(cpuprofileFile)
				if err != nil {
					log.Printf("Não foi possível iniciar o perfil da CPU%s\n", err)
				}
			}
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			pprof.StopCPUProfile()
			cpuprofileFile.Close()

			runtime.GC()
			if memoryprofile != "" {
				f, err := os.Create(memoryprofile)
				if err != nil {
					log.Printf("Não foi possível criar o arquivo de perfil de memória: %s\n", err)
					return
				}
				err = pprof.WriteHeapProfile(f)
				if err != nil {
					log.Printf("Não foi possível iniciar a criação de perfil de memória: %s\n", err)
				}
			}
		},
		Version: "0.2.4",
	}

	rootCmd.PersistentFlags().Int16VarP(&port, "porta", "P", 6666, "Especifique uma porta de câmera alternativa para se conectar")
	rootCmd.PersistentFlags().StringVarP(&username, "nome de usuário", "u", "admin", "Especifique o nome de usuário da câmera")
	rootCmd.PersistentFlags().StringVarP(&password, "senha", "p", "12345", "Especifique a senha da câmera")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "detalhe", "d", false, "Imprimir saída detalhada")
	rootCmd.PersistentFlags().StringVarP(&cpuprofile, "cpuprofile", "c", "", "Uso da CPU do perfil")
	rootCmd.PersistentFlags().StringVarP(&memoryprofile, "memoryprofile", "m", "", "Uso de memória do perfil")

	rootCmd.PersistentFlags().MarkHidden("cpuprofile")
	rootCmd.PersistentFlags().MarkHidden("memoryprofile")

	var ls = &cobra.Command{
		Use:   "ls [Cameras IP Address]",
		Short: "Listar arquivos armazenados no cartão SD da câmera",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			files, err := camera.GetFileList()
			if err != nil {
				log.Printf("ERRO Recebendo Lista de Arquivos: %s\n", err)
				return
			}

			for _, file := range files {
				fmt.Printf("%s\t%d\n", file.Path, file.Size)
			}
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var discover = &cobra.Command{
		Use:   "discover",
		Short: "Tente descobrir uma câmera enviando transmissões UDP",
		Args:  cobra.MaximumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			cameraIP, err := libipcamera.AutodiscoverCamera(verbose)
			if err != nil {
				log.Printf("ERRO ao descobrir a câmera: %s\n", err)
			}

			log.Printf("Câmera encontrada: %+v\n", cameraIP)
		},
	}

	var still = &cobra.Command{
		Use:   "still [Cameras IP Address]",
		Short: "Tire uma foto e salve no cartão SD",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			camera.TakePicture()
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var record = &cobra.Command{
		Use:   "record [Cameras IP Address]",
		Short: "Comece a gravar o vídeo no cartão SD",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			camera.StartRecording()
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var stop = &cobra.Command{
		Use:   "stop [Cameras IP Address]",
		Short: "Pare de gravar vídeo no cartão SD",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			camera.StopRecording()
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var firmware = &cobra.Command{
		Use:   "firmware [Cameras IP Address]",
		Short: "Recupere as informações da versão do firmware da câmera",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			firmware, err := camera.GetFirmwareInfo()
			if err != nil {
				log.Printf("ERRO ao recuperar informações da versão: %s\n", err)
				return
			}
			log.Printf("Versão do firmware: %s\n", firmware)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var rtsp = &cobra.Command{
		Use:   "rtsp [Cameras IP Address]",
		Short: "Inicie um RTSP-Server para visualização das câmeras.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			rtspServer := rtsp.CreateServer(applicationContext, "127.0.0.1", 8554, camera)
			defer rtspServer.Stop()

			log.Printf("Servidor RTSP criado\n")
			err := rtspServer.ListenAndServe()

			if err != nil {
				log.Printf("Servidor RTSP criado: %s\n", err)
			}
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var cmd = &cobra.Command{
		Use:   "cmd [RAW Command] [Cameras IP Address]",
		Short: "Envie um comando bruto para a câmera",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			command, err := hex.DecodeString(args[0])
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				return
			}

			if len(command) >= 2 {
				header := libipcamera.CreateCommandHeader(uint32(binary.BigEndian.Uint16(command[:2])))
				payload := command[2:]
				packet := libipcamera.CreatePacket(header, payload)
				log.Printf("Enviando comando: %X\n", packet)
				camera.SendPacket(packet)
			}

			log.Printf("Aguardando dados, pressione ENTER para sair")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[1]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	var fetch = &cobra.Command{
		Use:   "fetch [Cameras IP Address]",
		Short: "Baixe arquivos do cartão SD da câmera",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			files, err := camera.GetFileList()
			if err != nil {
				log.Printf("ERRO ao receber a lista de arquivos: %s\n", err)
				return
			}

			newestFile := files[len(files)-1].Path
			url := "http://" + args[0] + newestFile
			log.Printf("Baixando o arquivo mais recente: %s\n", url)
			downloadFile(filepath.Base(newestFile), url)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				camera = connectAndLogin(discoverCamera(verbose), int(port), username, password, verbose)
			} else {
				camera = connectAndLogin(net.ParseIP(args[0]), int(port), username, password, verbose)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			camera.Disconnect()
		},
	}

	rootCmd.AddCommand(ls)
	rootCmd.AddCommand(cmd)
	rootCmd.AddCommand(still)
	rootCmd.AddCommand(stop)
	rootCmd.AddCommand(fetch)
	rootCmd.AddCommand(record)
	rootCmd.AddCommand(firmware)
	rootCmd.AddCommand(rtsp)
	rootCmd.AddCommand(discover)

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func downloadFile(filepath string, url string) error {


	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()


	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()


	_, err = io.Copy(out, resp.Body)
	return err
}

func discoverCamera(verbose bool) net.IP {
	cameraIP, err := libipcamera.AutodiscoverCamera(verbose)
	if err != nil {
		log.Printf("ERRO durante a descoberta automática: %s\n", err)
	}
	if verbose {
		log.Printf("Câmera encontrada: %s\n", cameraIP)
	}
	return cameraIP
}
