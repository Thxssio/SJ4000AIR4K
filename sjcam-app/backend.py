import cv2 as cv
from PyQt5.QtGui import QImage
from PyQt5.QtCore import pyqtSignal,QThread



class backend(QThread):
    ImageUpdate = pyqtSignal(QImage)
    def run(self):
        cap=cv.VideoCapture(0)
        self.ThreadActive = True
        while self.ThreadActive:
            ret,frame=cap.read()
            if not ret:
                print("sem frame")
                cap.release()
                break
            frame = cv.cvtColor(frame, cv.COLOR_BGR2RGB)
            h, w, ch = frame.shape
            convertToQtFormat = QImage(frame, w, h, ch*w, QImage.Format_RGB888)
            self.ImageUpdate.emit(convertToQtFormat)
        cap.release()
        
            
        
    def stop(self):
        self.ThreadActive = False
        self.quit()
        

