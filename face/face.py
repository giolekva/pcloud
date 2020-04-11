from facenet_pytorch import MTCNN
import cv2
from PIL import Image, ImageDraw, ImageColor
import numpy as np
from matplotlib import pyplot as plt

mtcnn = MTCNN(keep_all=True)

img = Image.open("face.jpg")

boxes, _ = mtcnn.detect(img)
draw = ImageDraw.Draw(img)
for i, box in enumerate(boxes):
    draw.rectangle(((box[0], box[1]), (box[2], box[3])), outline="red")
img.save("detected.jpg")
#print(face)
