load("//:bazel_tools/docker.bzl", "docker_image")
load("//:bazel_tools/helm.bzl", "helm_install", "helm_uninstall")
load("@rules_pkg//:pkg.bzl", "pkg_tar")

# TODO(lekva): figure out how to build py_binary with pip dependencies and
# migrate off docker_image rule
docker_image(
	name = "push_to_dev",
	registry = "localhost:30500",
	image = "giolekva/pcloud-event-processor",
	tag = "latest",
	dockerfile = "Dockerfile",
	srcs = glob(["**"], exclude=["Dockerfile"]),
)

pkg_tar(
	name = "chart",
	srcs = glob(["chart/**"]),
	extension = "tar.gz",
	strip_prefix = "./chart",
)

helm_install(
	name = "install",
	namespace = "pcloud-event-processor",
	release_name = "init",
	chart = ":chart",
	args = {
	     "image.name": "localhost:30500/giolekva/pcloud-event-processor",
     	     "image.pullPolicy": "Always",
	},
)

helm_uninstall(
	name = "uninstall",
	namespace = "pcloud-event-processor",
	release_name = "init",
)
