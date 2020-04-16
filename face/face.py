import os
import sys

from facenet_pytorch import MTCNN, InceptionResnetV1
from PIL import Image


def detect(input_dir, output_dir):
    mtcnn = MTCNN(keep_all=True)
    resnet = InceptionResnetV1(pretrained='vggface2').eval()
    for f in os.listdir(input_dir):
        with Image.open(input_dir + "/" + f) as img:
            # if img.filename != "input/P7260028.jpg":
            #     continue
            print(img.filename)
            for m in mtcnn(img):
                print(resnet(m))
            
            # embedding = resnet(mtcnn(img))
            # print(len(embedding[0]))
            
            # boxes, _ = mtcnn.detect(img)
            # for i, box in enumerate(boxes):
            #     cropped = img.crop(box)
            #     cropped.save(output_dir + "/" + str(i) + "_" + f)


def classify(input_dir, output_dir):
    mtcnn = MTCNN()
    resnet = InceptionResnetV1(pretrained='vggface2').eval()
    for f in os.listdir(input_dir):
        with Image.open(input_dir + "/" + f) as img:
            print(img.filename)            
            embedding = resnet(mtcnn(img))
            print(len(embedding[0]))
    

def main():
    if sys.argv[1] == "detect":
        detect(sys.argv[2], sys.argv[3])
    else:
        classify(sys.argv[2], sys.argv[3])


if __name__ == "__main__":
    main()
