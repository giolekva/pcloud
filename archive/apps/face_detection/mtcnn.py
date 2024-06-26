from facenet_pytorch import MTCNN
from PIL import Image


def detect_faces(img_file):
    mtcnn = MTCNN(keep_all=True)
    ret = []
    with Image.open(img_file) as img:
        for box in mtcnn.detect(img)[0]:
            ret.append((box[0], box[1], box[2], box[3]))
    return ret
