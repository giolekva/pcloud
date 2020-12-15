load("//:bazel_tools/docker.bzl", "docker_image")
load("@rules_pkg//:pkg.bzl", "pkg_tar")

# TODO(lekva): figure out how to build py_binary with pip dependencies and
# migrate off docker_image rule
docker_image(
	name = "push_to_dev",
	registry = "localhost:30500",
	image = "giolekva/face-detector",
	tag = "latest",
	dockerfile = "Dockerfile",
	srcs = glob(["*.py"]) + [
	     "haarcascade_frontalface_default.xml",
	],
)

pkg_tar(
	name = "chart",
	srcs = glob(["chart/**"]),
	extension = "tar.gz",
	strip_prefix = "./chart",
)
