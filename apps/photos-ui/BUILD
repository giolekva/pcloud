load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_image", "container_push")
load("@rules_pkg//:pkg.bzl", "pkg_tar")

filegroup(
    name = "data",
    srcs = glob([
        "static/**",
        "*.html",
    ]),
)

go_binary(
    name = "photos_ui",
    srcs = ["main.go"],
    data = [":data"],
    deps = [
        "@io_bazel_rules_go//go/tools/bazel",
    ],
)

container_image(
    name = "container",
    base = "@alpine_base//image",
    cmd = ["/photos_ui"],
    data_path = "/",
    env = {
        "RUNFILES_DIR": "/",
        "RUNFILES_MANIFEST_FILE": "/photos_ui.runfiles_manifest",
    },
    files = [
        "photos_ui.runfiles_manifest",
        ":data",
        ":photos_ui",
    ],
    symlinks = {
        "/photos_ui": "/apps/photos-ui/photos_ui_/photos_ui",
    },
)

container_push(
    name = "push_to_dev",
    format = "Docker",
    image = ":container",
    registry = "localhost:30500",
    repository = "giolekva/photos-ui",
    tag = "latest",
)

pkg_tar(
    name = "chart",
    srcs = glob(["chart/**"]),
    extension = "tar.gz",
    strip_prefix = "./chart",
)
