load("@rules_foreign_cc//foreign_cc:defs.bzl", "cmake")

cmake(
    name = "evmc",
    lib_source = "@evmc//:all_srcs",
    out_headers_only = True,
    visibility = ["//visibility:public"],
)
