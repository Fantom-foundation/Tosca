include_guard(GLOBAL)


option(TOSCA_COVERAGE "Enable coverage report for evmzero." OFF)

if(TOSCA_COVERAGE)
  if (NOT CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
    # gcc uses gcov, clang should use it as well, but it has version compatibility issues
    # between compiler generate code and the gcov tool.
    message(FATAL_ERROR "Coverage is only supported with GCC.")
  endif()

  add_compile_options($<$<CXX_COMPILER_ID:GNU>:--coverage>)
  add_link_options($<$<CXX_COMPILER_ID:GNU>:--coverage>)

  find_program(LCOV lcov REQUIRED)
  find_program(GENHTML genhtml REQUIRED)

  add_custom_target(coverage
    COMMENT "Generating coverage report."
    COMMAND ${LCOV} 
      --capture 
      --directory .  
      "$<$<VERSION_GREATER_EQUAL:${CMAKE_CXX_COMPILER_VERSION},13.2.0>:--ignore-errors>" 
      "$<$<VERSION_GREATER_EQUAL:${CMAKE_CXX_COMPILER_VERSION},13.2.0>:mismatch,mismatch,gcov>"
      --output-file coverage.info
    COMMAND ${LCOV} 
      --remove coverage.info 
      '/usr/include/*' 
      '/usr/lib/*' 
      '*/third_party/*'
      --output-file filtered.info
    COMMAND ${GENHTML} filtered.info --output-directory coverage
    WORKING_DIRECTORY ${CMAKE_BINARY_DIR}
  )
endif()