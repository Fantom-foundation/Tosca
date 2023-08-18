cc_library(
    name = "evmzero",
    srcs = [
        "allocator.cc",
        "evmzero.cc",
        "interpreter.cc",
        "memory.cc",
        "opcodes.cc",
        "stack.cc",
    ],
    hdrs = glob([
        "*.h",
        "*.inc",
    ]),
    deps = [
        "//common",
        "//third_party/ethash",
        "//third_party/evmc",
        "//third_party/intx",
        "@absl//absl/container:flat_hash_map",
    ],
)

cc_test(
    name = "interpreter_test",
    srcs = ["interpreter_test.cc"],
    deps = [
        ":evmzero",
        "@googletest//:gtest_main",
    ],
)

cc_test(
    name = "memory_test",
    srcs = ["memory_test.cc"],
    deps = [
        ":evmzero",
        "@googletest//:gtest_main",
    ],
)

cc_test(
    name = "profiler_test",
    srcs = ["profiler_test.cc"],
    deps = [
        ":evmzero",
        "@googletest//:gtest_main",
    ],
)

cc_test(
    name = "stack_test",
    srcs = ["stack_test.cc"],
    deps = [
        ":evmzero",
        "@googletest//:gtest_main",
    ],
)

cc_test(
    name = "uint256_test",
    srcs = ["uint256_test.cc"],
    deps = [
        ":evmzero",
        "@googletest//:gtest_main",
    ],
)

cc_binary(
    name = "sha3_cache_benchmark",
    srcs = ["sha3_cache_benchmark.cc"],
    deps = [
        ":evmzero",
        "@googlebenchmark//:benchmark_main",
    ],
)
