cmake_minimum_required(VERSION 3.12)
project(tosca)

option(TOSCA_ASAN "Enable AddressSanitizer for Tosca targets.")
option(TOSCA_ASSERT "Enable Tosca specific assertions.")

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
enable_testing()

set(TOSCA_THIRD_PARTY_DIR ${PROJECT_SOURCE_DIR}/../third_party)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/evmc third_party/evmc EXCLUDE_FROM_ALL)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/ethash third_party/ethash EXCLUDE_FROM_ALL)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/googletest third_party/googletest EXCLUDE_FROM_ALL)

function(tosca_add_compile_flags target)
  set_target_properties(${target} PROPERTIES
    CXX_STANDARD 20
    CXX_STANDARD_REQUIRED ON
    CXX_EXTENSIONS OFF)
  target_compile_options(${target} PRIVATE
    $<$<CXX_COMPILER_ID:MSVC>:/MP /W4 /external:W0>
    $<$<CXX_COMPILER_ID:GNU,Clang,AppleClang>:-Wall -Wextra -Wpedantic -Wtype-limits -Wconversion -Wsign-conversion -Wdouble-promotion -g>
    $<$<CXX_COMPILER_ID:GNU>:-Wno-missing-field-initializers -fdiagnostics-color>
    $<$<CXX_COMPILER_ID:Clang,AppleClang>:-fcolor-diagnostics>)
  target_compile_definitions(${target} PRIVATE $<$<BOOL:${TOSCA_ASSERT}>:TOSCA_ASSERT_ENABLED>)
  target_include_directories(${target} PRIVATE ${PROJECT_SOURCE_DIR})
endfunction()

if(TOSCA_ASAN)
  add_compile_options($<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
  add_link_options($<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
endif()

add_subdirectory(common)
add_subdirectory(vm/evmzero)
