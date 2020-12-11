load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_push", "container_image")
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
    files = [":rpuppy"],
    cmd = ["rpuppy"],
)

container_push(
   name = "push_to_dev",
   image = ":container",
   format = "Docker",
   registry = "localhost:30500",
   repository = "giolekva/rpuppy-arm",
   tag = "latest",
)

pkg_tar(
	name = "chart",
	srcs = glob(["chart/**"]),
	extension = "tar.gz",
	strip_prefix = "./chart",
)
