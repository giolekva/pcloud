import sys
import json
import urllib.parse
import urllib.request
import os


def fetch_file_for_image(gql_endpoint, object_storage_endpoint, id):
    data = {"query": "{ getImage(id: \"" + id + "\") { objectPath } }"}
    # encoded_data = urllib.parse.urlencode(data).encode('UTF-8')
    req = urllib.request.Request(gql_endpoint, method="POST")
    req.add_header('Content-Type', 'application/json')
    resp = urllib.request.urlopen(req, json.dumps(data).encode('UTF-8'))
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
    data = {"query": "mutation {{ addImageSegment(input: [{segments}]) {{ imageSegment {{ id }} }} }}".format(
        segments=", ".join(segments))}
    # encoded_data = urllib.parse.urlencode(data).encode('UTF-8')
    req = urllib.request.Request(gql_endpoint, method="POST")
    req.add_header('Content-Type', 'application/json')
    resp = urllib.request.urlopen(req, json.dumps(data).encode('UTF-8'))
    print(resp.read())
    

def main():
    method = "haar"
    if len(sys.argv) == 5 and sys.argv[4] == "mtcnn":
        method = "mtcnn"
    f = fetch_file_for_image(sys.argv[1], sys.argv[2], sys.argv[3])
    if method == "haar":
        import haar
        faces = haar.detect_faces(f)
        upload_face_segments(sys.argv[1], sys.argv[3], faces)
    else:
        import mtcnn
        faces = mtcnn.detect_faces(f)
        upload_face_segments(sys.argv[1], sys.argv[3], faces)
    os.remove(f)


if __name__ == "__main__":
    main()
