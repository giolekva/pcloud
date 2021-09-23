build:
	make -C nebula build/linux-arm64/nebula
	make -C nebula build/linux-arm64/nebula-cert

clean:
	rm -f nebula/build

image: clean build
	docker build --tag=giolekva/nebula .

push: image
	docker push giolekva/nebula:latest
