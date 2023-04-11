# This directory contains common definitions and components of general utility
# shared among multiple VM implementations placed in sibling directories.

cc_library(
    name = "word",
    srcs = ["word.cc"],
    hdrs = ["word.h"],
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_calccrypto_uint256_t//:uint256_t",
    ],
)

cc_test(
    name = "word_test",
    srcs = ["word_test.cc"],
    deps = [
        ":word",
        "@com_google_googletest//:gtest_main",
    ],
)
