import subprocess
import os
import tkinter
from tkinter import *


def iniciar_camera():
    args = "CamOpen.exe rtsp 192.168.100.1"
    subprocess.call(args, shell=True)

if __name__ == '__main__':
    window = Tk()
    window.title("AD CANUDOS - CAMERAS")
    window.config(padx=10, pady=100)
    text = Label(window, text="rtsp://127.0.0.1:8554")
    text.place(x=70,y=90)
    label1 = Label(window, text="Helvetica", font=("Helvetica", 18))
    add_button = Button(text="INICIAR CAMERA - SJCAM", width=36, command=iniciar_camera)
    add_button.grid(row=4, column=1, columnspan=2)


    window.mainloop()

