load("@rules_pkg//:pkg.bzl", "pkg_tar")

pkg_tar(
    name = "chart",
    srcs = glob(["chart/**"]),
    extension = "tar.gz",
    strip_prefix = "./chart",
)
