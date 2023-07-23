import wmi

ti = 0
name = 'CamOpen.exe'
f = wmi.WMI()

for process in f.win32_Process():
    if process.name == name:
        process.Terminate()
        ti +=1
if ti==0:
    print("Processo não encontrado.")