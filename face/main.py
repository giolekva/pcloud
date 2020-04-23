import sys
import json
import urllib.parse
import urllib.request
import os

from facenet_pytorch import MTCNN, InceptionResnetV1
from PIL import Image


def detect_faces(img_file):
    mtcnn = MTCNN(keep_all=True)
    ret = []
    with Image.open(img_file) as img:
        for box in mtcnn.detect(img)[0]:
            ret.append((box[0], box[1], box[2], box[3]))
    return ret


def fetch_file_for_image(gql_endpoint, object_storage_endpoint, id):
    data = {"query": "{ getImage(id: \"" + id + "\") { objectPath } }"}
    encoded_data = urllib.parse.urlencode(data).encode('UTF-8')
    req = urllib.request.Request(gql_endpoint, encoded_data, method="POST")
    resp = urllib.request.urlopen(req)
    object_path = json.loads(resp.read())["getImage"]["objectPath"]
    local_path = urllib.request.urlretrieve(
        object_storage_endpoint + "/" + object_path)[0]
    return local_path


def format_img_segment(id, box):
    return ("{{upperLeftX: {f[0]}, upperLeftY: {f[1]}, lowerRightX: {f[2]}, " +
            "lowerRightY: {f[3]}, sourceImage: {{id: \"{id}\"}}}}").format(
                f=box,
                id=id)


def upload_face_segments(gql_endpoint, id, faces):
    segments = [format_img_segment(id, f) for f in faces]
    data = {"query": "mutation {{ addImageSegment(input: [{segments}]) {{ imagesegment {{ id }} }} }}".format(
        segments=", ".join(segments))}
    encoded_data = urllib.parse.urlencode(data).encode('UTF-8')
    req = urllib.request.Request(gql_endpoint, encoded_data, method="POST")
    resp = urllib.request.urlopen(req)
    print(resp.read())
    

def main():
    f = fetch_file_for_image(sys.argv[1], sys.argv[2], sys.argv[3])
    faces = detect_faces(f)
    os.remove(f)
    upload_face_segments(sys.argv[1], sys.argv[3], faces)


if __name__ == "__main__":
    main()
