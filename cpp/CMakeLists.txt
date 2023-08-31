cmake_minimum_required(VERSION 3.12)
set(CMAKE_POLICY_DEFAULT_CMP0077 NEW) # option respects set variables
project(tosca)

option(TOSCA_ASAN "Enable AddressSanitizer for Tosca targets.")
option(TOSCA_ASSERT "Enable Tosca specific assertions.")

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_POSITION_INDEPENDENT_CODE TRUE)
enable_testing()

if(TOSCA_ASAN)
  add_compile_options(
    $<$<CXX_COMPILER_ID:Clang>:-fno-omit-frame-pointer>
    $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
  add_link_options(
    $<$<CXX_COMPILER_ID:Clang>:-fno-omit-frame-pointer>
    $<$<CXX_COMPILER_ID:Clang>:-fsanitize=address>)
endif()

set(TOSCA_THIRD_PARTY_DIR ${PROJECT_SOURCE_DIR}/../third_party)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/evmc third_party/evmc EXCLUDE_FROM_ALL)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/ethash third_party/ethash EXCLUDE_FROM_ALL)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/googletest third_party/googletest EXCLUDE_FROM_ALL)

set(MI_OVERRIDE OFF)
set(MI_BUILD_TESTS OFF)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/mimalloc third_party/mimalloc EXCLUDE_FROM_ALL)

set(BENCHMARK_ENABLE_TESTING OFF)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/google_benchmark third_party/google_benchmark EXCLUDE_FROM_ALL)

if (NOT ${CMAKE_SYSTEM_NAME} MATCHES "Darwin")
  set(BUILD_TESTING OFF)
  add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/gperftools third_party/gperftools EXCLUDE_FROM_ALL)
endif()

set(ABSL_PROPAGATE_CXX_STD ON)
set(ABSL_USE_SYSTEM_INCLUDES ON)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/abseil-cpp third_party/abseil-cpp EXCLUDE_FROM_ALL)

set(TRACY_STATIC OFF)
add_subdirectory(${TOSCA_THIRD_PARTY_DIR}/tracy third_party/tracy EXCLUDE_FROM_ALL)

function(tosca_add_compile_flags target)
  set_target_properties(${target} PROPERTIES
    CXX_STANDARD 20
    CXX_STANDARD_REQUIRED ON
    CXX_EXTENSIONS OFF)
  target_compile_options(${target} PRIVATE
    $<$<CXX_COMPILER_ID:MSVC>:/MP /W4 /external:W0>
    $<$<CXX_COMPILER_ID:GNU,Clang,AppleClang>:-march=native -mtune=native -Wall -Wextra -Wpedantic -Wtype-limits -Wconversion -Wsign-conversion -Wdouble-promotion -Wno-gnu-label-as-value -g>
    $<$<CXX_COMPILER_ID:GNU>:-Wno-missing-field-initializers -fdiagnostics-color>
    $<$<CXX_COMPILER_ID:Clang,AppleClang>:-fcolor-diagnostics>)
  target_compile_definitions(${target} PRIVATE $<$<BOOL:${TOSCA_ASSERT}>:TOSCA_ASSERT_ENABLED>)
  target_include_directories(${target} PRIVATE ${PROJECT_SOURCE_DIR})
endfunction()

add_subdirectory(common)
add_subdirectory(vm/evmzero)
