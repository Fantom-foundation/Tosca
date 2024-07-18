include_guard(GLOBAL)

option(TOSCA_COVERAGE "Enable coverage report for evmzero." OFF)

if(TOSCA_COVERAGE)
 
  if (NOT CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
    # gcc uses gcov, clang should use it as well, but it has version compatibility issues
    # between compiler generate code and the gcov tool.
    message(FATAL_ERROR "Coverage build currently is only supported with GCC.")
  endif()

  add_compile_options(--coverage)
  add_link_options(--coverage)
  add_definitions(-DTOSCA_COVERAGE=1)

  find_program(LCOV lcov REQUIRED)
  find_program(GCOVR gcovr REQUIRED)
  find_program(GENHTML genhtml REQUIRED)

  add_custom_target(coverage
    COMMENT "Generating coverage report."

    # Capture coverage data
    COMMAND ${LCOV} 
      --capture 
      --directory .  
      "$<$<VERSION_GREATER_EQUAL:${CMAKE_CXX_COMPILER_VERSION},13.2.0>:--ignore-errors>" 
      "$<$<VERSION_GREATER_EQUAL:${CMAKE_CXX_COMPILER_VERSION},13.2.0>:mismatch,mismatch,gcov>"
      --output-file coverage.info

    # filter coverage data
    COMMAND ${LCOV} 
      --remove coverage.info 
      '/usr/include/*' 
      '/usr/lib/*' 
      '*/third_party/*'
      --output-file filtered.info

    # build and compress HTML report
    COMMAND ${GENHTML} filtered.info --output-directory coverage --parallel
    COMMAND tar czvf coverage.tar.gz coverage/ 

    # Text report
    COMMAND ${GCOVR} -r .. | tee coverage.txt

    WORKING_DIRECTORY ${CMAKE_BINARY_DIR}
  )

  # Add a target to clean coverage data (will print found data coverage report)
  # executed on demand
  add_custom_target(clean_coverage_data
    COMMAND ${GCOVR} -r .. 
    COMMAND ${CMAKE_COMMAND} -E remove -f coverage.txt coverage coverage.tar.gz
  )

endif()