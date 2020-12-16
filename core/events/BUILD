load("//:bazel_tools/docker.bzl", "docker_image")
load("//:bazel_tools/helm.bzl", "helm_install", "helm_uninstall")
load("@rules_pkg//:pkg.bzl", "pkg_tar")

# TODO(lekva): figure out how to build py_binary with pip dependencies and
# migrate off docker_image rule
docker_image(
    name = "push_to_dev",
    srcs = glob(
        ["**"],
        exclude = ["Dockerfile"],
    ),
    dockerfile = "Dockerfile",
    image = "giolekva/pcloud-event-processor",
    registry = "localhost:30500",
    tag = "latest",
)

pkg_tar(
    name = "chart",
    srcs = glob(["chart/**"]),
    extension = "tar.gz",
    strip_prefix = "./chart",
)

helm_install(
    name = "install",
    args = {
        "image.name": "localhost:30500/giolekva/pcloud-event-processor",
        "image.pullPolicy": "Always",
    },
    chart = ":chart",
    namespace = "pcloud-event-processor",
    release_name = "init",
)

helm_uninstall(
    name = "uninstall",
    namespace = "pcloud-event-processor",
    release_name = "init",
)
