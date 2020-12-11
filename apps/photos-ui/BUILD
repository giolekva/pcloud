load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_push", "container_image")
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
	     "@io_bazel_rules_go//go/tools/bazel:bazel",
	],
)

container_image(
    name = "container",
    base = "@alpine_base//image",
    files = [
    	  ":photos_ui",
	  ":data",
	  "photos_ui.runfiles_manifest",
    ],
    data_path = "/",
    cmd = ["/photos_ui"],
    env = {
    	"RUNFILES_DIR": "/",
    	"RUNFILES_MANIFEST_FILE": "/photos_ui.runfiles_manifest",
    },
    symlinks = {
    	     "/photos_ui": "/apps/photos-ui/photos_ui_/photos_ui",
    },
)

container_push(
   name = "push_to_dev",
   image = ":container",
   format = "Docker",
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
