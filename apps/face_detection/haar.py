import cv2

def detect_faces(img_file):
    face_cascade = cv2.CascadeClassifier('haarcascade_frontalface_default.xml')
    img = cv2.imread(img_file)
    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    faces = face_cascade.detectMultiScale(gray, 1.1, 4)
    ret = []
    for (x, y, w, h) in faces:
        ret.append((x, y, x + w, y + h))
    return ret
