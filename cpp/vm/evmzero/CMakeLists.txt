option(EVMZERO_MIMALLOC "Enable mimalloc for evmzero" ON)

###########################################################
# Library
add_library(evmzero SHARED
  evmzero.cc
  allocator.cc
  interpreter.cc
  memory.cc
  opcodes.cc
  stack.cc)
tosca_add_compile_flags(evmzero)
target_include_directories(evmzero SYSTEM PUBLIC ${TOSCA_THIRD_PARTY_DIR}/intx/include)
target_link_libraries(evmzero PUBLIC ethash::keccak evmc::evmc)

if(EVMZERO_MIMALLOC)
  target_compile_definitions(evmzero PRIVATE EVMZERO_ENABLE_MIMALLOC)
  target_link_libraries(evmzero PRIVATE mimalloc-obj)
endif()

###########################################################
# Tests
include(GoogleTest)

file(GLOB_RECURSE evmzero_tests CONFIGURE_DEPENDS *_test.cc)
foreach(evmzero_test ${evmzero_tests})
  get_filename_component(evmzero_test_target ${evmzero_test} NAME_WE)

  add_executable(${evmzero_test_target} ${evmzero_test})
  tosca_add_compile_flags(${evmzero_test_target})
  target_link_libraries(${evmzero_test_target} evmzero GTest::gmock GTest::gtest_main)
  gtest_discover_tests(${evmzero_test_target})

  # Always enable ASAN for unit tests
  target_compile_options(${evmzero_test_target} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
  target_link_options(${evmzero_test_target} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
endforeach()

###########################################################
# Benchmarks

file(GLOB_RECURSE evmzero_benchmarks CONFIGURE_DEPENDS *_benchmark.cc)
foreach(evmzero_benchmark ${evmzero_benchmarks})
  get_filename_component(evmzero_benchmark_target ${evmzero_benchmark} NAME_WE)

  add_executable(${evmzero_benchmark_target} ${evmzero_benchmark})
  tosca_add_compile_flags(${evmzero_benchmark_target})
  target_link_libraries(${evmzero_benchmark_target} evmzero benchmark::benchmark benchmark::benchmark_main)

  # Always include the CPU profiler in benchmarks.
  if (NOT ${CMAKE_SYSTEM_NAME} MATCHES "Darwin")
    target_link_libraries(${evmzero_benchmark_target} -Wl,--no-as-needed profiler  -Wl,--as-needed)
  endif()
endforeach()
