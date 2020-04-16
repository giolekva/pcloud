FROM python:3

RUN pip help
RUN pip install torch --no-cache-dir
RUN pip install torchvision --no-cache-dir
RUN pip install facenet-pytorch --no-cache-dir
RUN pip install opencv-python-headless --no-cache-dir
RUN pip install matplotlib --no-cache-dir

# COPY . /face
WORKDIR /face

# RUN rm -r -f output/*
CMD python face.py detect input output
# CMD python face.py classify output class
