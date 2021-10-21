FROM golang:1.17.2 AS build

WORKDIR /code
RUN git clone https://git.sr.ht/~migadu/alps
WORKDIR /code/alps
RUN go mod download

ENV CGO_ENABLED=0
ENV GO111MODULE=on
RUN go build -o alps cmd/alps/main.go

FROM alpine:3.14.2

WORKDIR /
COPY --from=build /code/alps ./alps
# COPY --from=build /code/alps/alps ./alps
# RUN chmod +x /alps/alps
# COPY --from=build /code/alps/themes ./themes
