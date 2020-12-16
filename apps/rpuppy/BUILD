load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_image", "container_push")
load("@rules_pkg//:pkg.bzl", "pkg_tar")

go_binary(
    name = "rpuppy",
    srcs = ["main.go"],
    deps = [
        "@com_github_itaysk_regogo//:go_default_library",
    ],
)

container_image(
    name = "container",
    # References container_pull from WORKSPACE (above)
    base = "@alpine_base//image",
    cmd = ["rpuppy"],
    files = [":rpuppy"],
)

container_push(
    name = "push_to_dev",
    format = "Docker",
    image = ":container",
    registry = "localhost:30500",
    repository = "giolekva/rpuppy",
    tag = "latest",
)

pkg_tar(
    name = "chart",
    srcs = glob(["chart/**"]),
    extension = "tar.gz",
    strip_prefix = "./chart",
)
