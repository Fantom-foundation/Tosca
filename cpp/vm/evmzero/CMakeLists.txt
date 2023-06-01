###########################################################
# Library
add_library(evmzero SHARED
  evmzero.cc
  interpreter.cc
  memory.cc
  stack.cc)
tosca_add_compile_flags(evmzero)
target_include_directories(evmzero SYSTEM PUBLIC ${TOSCA_THIRD_PARTY_DIR}/intx/include)
target_link_libraries(evmzero PUBLIC ethash::keccak evmc::evmc)

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
