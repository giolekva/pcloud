image_arm64:
	docker build --file=Dockerfile.flux --tag=giolekva/flux:latest . --platform=linux/arm64

push_arm64: image_arm64
	docker push giolekva/flux:latest
