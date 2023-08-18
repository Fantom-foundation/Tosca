load("@rules_foreign_cc//foreign_cc:defs.bzl", "cmake")

package(
    features = ["external_include_paths"],
)

cmake(
    name = "intx",
    cache_entries = {
        "INTX_TESTING": "OFF",
    },
    lib_source = "@intx//:all_srcs",
    out_headers_only = True,
    visibility = ["//visibility:public"],
)
