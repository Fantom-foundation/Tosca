load("@rules_foreign_cc//foreign_cc:defs.bzl", "cmake")

package(
    features = ["external_include_paths"],
)

cmake(
    name = "ethash",
    lib_source = "@ethash//:all_srcs",
    out_static_libs = ["libkeccak.a"],
    visibility = ["//visibility:public"],
    alwayslink = 1,
)
