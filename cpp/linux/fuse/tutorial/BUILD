load("@rules_cc//cc:defs.bzl", "cc_binary")

cc_binary(
    name = "main",
    srcs = ["main.cpp"],
    deps = ["@com_github_google_glog//:glog"],
    copts = ["-I/usr/include/fuse3", "-D_FILE_OFFSET_BITS=64", "-DFUSE_USE_VERSION=311"],
    linkopts = ["-lfuse3", "-lpthread"],
    visibility = ["//visibility:public"],
)
