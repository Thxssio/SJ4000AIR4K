import subprocess
from tkinter import *
import wmi


codloop = True


def switch(sjcam):
    global processcam
    ti = 0
    name = 'CamOpen.exe'
    f = wmi.WMI()
    args = 'CamOpen.exe rtsp 192.168.100.1'

    if sjcam == "Iniciar":
        cam = subprocess.Popen(args, shell=True)
        print("Process ID:", cam.pid)
        processcam = cam.pid

    if sjcam == "Parar":
        for process in f.win32_Process():
            if process.name == name:
                process.Terminate()
                print("Processo encontrado!! Encerrando...")
                ti +=1
            print("Encerrado!")
        if ti==0:
            print("Processo não encontrado.")


     

def scan():
    if codloop:
        print("rodando")
    window.after(1000, scan)

def iniciar_camera():
    global codloop
    switch("Iniciar")
    codloop = True
    

def parar_camera():
    global codloop
    codloop = False
    switch("Parar")



if __name__ == '__main__':
    window = Tk()
    p1 = PhotoImage(file= 'icon1.png')
    window.iconphoto(False, p1)
    window.title("AD CANUDOS")
    window.config(padx=10, pady=50)

    window.geometry("300x200")
    frame = Frame(window, width=40, height=40)
    frame.pack()
    frame.place(anchor='center', relx=0.5, rely=0.5)
    text = Label(window, text="rtsp://127.0.0.1:8554", anchor='center')
    text.place(x=80,y=110)
    text = Label(window, text="@thxssio | SoftwareEnginner", anchor='center')
    text.place(x=60,y=130)
   

    button1 = Button(text="INICIAR CAMERA - SJCAM", width=20, command=iniciar_camera)
    button1.grid(row=4, column=1, columnspan=2, pady=10, padx=65)
    button2 = Button(text="PARAR CAMERA - SJCAM", width=20, command=parar_camera)
    button2.grid(row=6, column=1, columnspan=3, padx=65 )
    window.after(1000, scan)
    window.mainloop()

