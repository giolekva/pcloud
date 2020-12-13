load("@rules_pkg//:pkg.bzl", "pkg_tar")

# TODO(lekva): figure out how to build py_binary with pip dependencies and
# migrate off of shell script running docker build
sh_binary(
	name = "push_to_dev",
	srcs = ["push_to_dev.sh"],
	data = glob(["*.py"]) + [
	     "Dockerfile",
	     "haarcascade_frontalface_default.xml",
	],
	deps = [
	     "@bazel_tools//tools/bash/runfiles",
	],
)

pkg_tar(
	name = "chart",
	srcs = glob(["chart/**"]),
	extension = "tar.gz",
	strip_prefix = "./chart",
)
