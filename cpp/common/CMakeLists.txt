add_subdirectory(coverage)

###########################################################
# Library
add_library(tosca_common INTERFACE)
target_link_libraries(tosca_common INTERFACE absl::flat_hash_map)

###########################################################
# Tests
include(GoogleTest)

file(GLOB_RECURSE tosca_common_tests CONFIGURE_DEPENDS *_test.cc)
foreach(tosca_common_test ${tosca_common_tests})
  get_filename_component(tosca_common_test_target ${tosca_common_test} NAME_WE)

  add_executable(${tosca_common_test_target} ${tosca_common_test})
  tosca_add_compile_flags(${tosca_common_test_target})
  target_link_libraries(${tosca_common_test_target} tosca_common GTest::gmock GTest::gtest_main)
  gtest_discover_tests(${tosca_common_test_target})
endforeach()

###########################################################
# Benchmarks

file(GLOB_RECURSE tosca_common_benchmarks CONFIGURE_DEPENDS *_benchmark.cc)
foreach(tosca_common_benchmark ${tosca_common_benchmarks})
  get_filename_component(tosca_common_benchmark_target ${tosca_common_benchmark} NAME_WE)

  add_executable(${tosca_common_benchmark_target} ${tosca_common_benchmark})
  tosca_add_compile_flags(${tosca_common_benchmark_target})
  target_link_libraries(${tosca_common_benchmark_target} tosca_common benchmark::benchmark benchmark::benchmark_main)

  if (NOT ${CMAKE_SYSTEM_NAME} MATCHES "Darwin")
    # Always include the CPU profiler in benchmarks.
    target_link_libraries(${tosca_common_benchmark_target} -Wl,--no-as-needed profiler -Wl,--as-needed)
  endif()
endforeach()
