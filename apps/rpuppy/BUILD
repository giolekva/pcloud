load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_binary(
	name = "rpuppy",
	srcs = ["main.go"],
	deps = [
	     "@com_github_itaysk_regogo//:go_default_library",
	],
)

go_image(
	name = "rpuppy_image",
	binary = ":rpuppy",
)
