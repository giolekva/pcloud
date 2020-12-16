load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("@io_bazel_rules_docker//container:container.bzl", "container_image", "container_push")
load("@rules_pkg//:pkg.bzl", "pkg_tar")

go_library(
    name = "lib",
    srcs = ["importer.go"],
    importpath = "github.com/giolekva/pcloud/apps/minio/importer",
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@com_github_itaysk_regogo//:go_default_library",
    ],
)

go_binary(
    name = "importer",
    srcs = ["main.go"],
    deps = [
        ":lib",
    ],
)

container_image(
    name = "container",
    base = "@alpine_base//image",
    cmd = ["importer"],
    files = [":importer"],
)

container_push(
    name = "push_to_dev",
    format = "Docker",
    image = ":container",
    registry = "localhost:30500",
    repository = "giolekva/minio-importer",
    tag = "latest",
)

pkg_tar(
    name = "chart",
    srcs = glob(["chart/**"]),
    extension = "tar.gz",
    strip_prefix = "./chart",
)
