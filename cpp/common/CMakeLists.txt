include(GoogleTest)

file(GLOB_RECURSE tosca_common_tests CONFIGURE_DEPENDS *_test.cc)
foreach(tosca_common_test ${tosca_common_tests})
  get_filename_component(tosca_common_test_target ${tosca_common_test} NAME_WE)

  add_executable(${tosca_common_test_target} ${tosca_common_test})
  tosca_add_compile_flags(${tosca_common_test_target})
  target_link_libraries(${tosca_common_test_target} GTest::gmock GTest::gtest_main)
  gtest_discover_tests(${tosca_common_test_target})

  # Always enable ASAN for unit tests
  target_compile_options(${tosca_common_test_target} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
  target_link_options(${tosca_common_test_target} PRIVATE $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
endforeach()
