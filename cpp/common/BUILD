cc_library(
    name = "common",
    hdrs = glob(["*.h"]),
    include_prefix = "common",
    visibility = ["//visibility:public"],
)

cc_test(
    name = "lru_cache_test",
    srcs = ["lru_cache_test.cc"],
    deps = [
        ":common",
        "@googletest//:gtest_main",
    ],
)
