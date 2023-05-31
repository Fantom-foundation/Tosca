###########################################################
# Library
add_library(evmzero SHARED evmzero.cc)
tosca_add_compile_flags(evmzero)
target_include_directories(evmzero SYSTEM PUBLIC ${TOSCA_THIRD_PARTY_DIR}/intx/include)
target_link_libraries(evmzero PUBLIC ethash::keccak evmc::evmc)

###########################################################
# Tests
include(GoogleTest)

set(evmzero_tests uint256_test)
foreach(evmzero_test ${evmzero_tests})
  add_executable(${evmzero_test} ${evmzero_test}.cc)
  tosca_add_compile_flags(${evmzero_test})
  target_link_libraries(${evmzero_test} evmzero GTest::gmock GTest::gtest_main)
  gtest_discover_tests(${evmzero_test})

  # Always enable ASAN for unit tests
  target_compile_options(${evmzero_test} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
  target_link_options(${evmzero_test} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
endforeach()
