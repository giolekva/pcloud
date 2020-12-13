# TODO(lekva): figure out how to build py_binary with pip dependencies and
# migrate off of shell script running docker build
sh_binary(
	name = "container",
	srcs = ["container.sh"],
	data = glob(["*.py"]) + [
	     "Dockerfile",
	     "haarcascade_frontalface_default.xml",
	],
	deps = [
	     "@bazel_tools//tools/bash/runfiles",
	],
)
