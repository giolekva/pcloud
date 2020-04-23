FROM python:3

RUN pip install torch --no-cache-dir
RUN pip install torchvision --no-cache-dir
RUN pip install facenet-pytorch --no-cache-dir
RUN pip install opencv-python-headless --no-cache-dir
RUN pip install matplotlib --no-cache-dir

WORKDIR /src/go/src/github.com/giolekva/pcloud/face
COPY .  /src/go/src/github.com/giolekva/pcloud/face