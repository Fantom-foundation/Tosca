###########################################################
# Library
file(GLOB_RECURSE evmzero_srcs CONFIGURE_DEPENDS src/*.cc)
add_library(evmzero SHARED ${evmzero_srcs})
tosca_add_compile_flags(evmzero)
target_include_directories(evmzero PUBLIC src)
target_include_directories(evmzero SYSTEM PUBLIC ${TOSCA_THIRD_PARTY_DIR}/intx/include)
target_link_libraries(evmzero PUBLIC tosca_common ethash::keccak evmc::evmc)

###########################################################
# Tests
include(GoogleTest)

file(GLOB_RECURSE evmzero_tests CONFIGURE_DEPENDS tests/*.cc)
foreach(evmzero_test ${evmzero_tests})
  get_filename_component(evmzero_test_target ${evmzero_test} NAME_WE)

  add_executable(${evmzero_test_target} ${evmzero_test})
  tosca_add_compile_flags(${evmzero_test_target})
  target_link_libraries(${evmzero_test_target} evmzero GTest::gtest_main)
  gtest_discover_tests(${evmzero_test_target})

  # Always enable ASAN for unit tests
  target_compile_options(${evmzero_test_target} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
  target_link_options(${evmzero_test_target} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
endforeach()
