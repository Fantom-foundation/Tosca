// Wrappers around weak llvm coverage related symbols to get weak linkage on
// stable Rust.
#include <stddef.h>
#include <stdint.h>

void __llvm_profile_set_filename(const char *filename) __attribute__((weak));
void __llvm_profile_write_file(void) __attribute__((weak));
void __llvm_profile_reset_counters(void) __attribute__((weak));

uint8_t llvm_profile_enabled_wrapper() {
  return __llvm_profile_set_filename != NULL;
}

void llvm_profile_set_filename_wrapper(const char *filename) {
  if (__llvm_profile_set_filename) {
    __llvm_profile_set_filename(filename);
  }
}

void llvm_profile_write_file_wrapper(void) {
  if (__llvm_profile_write_file) {
    __llvm_profile_write_file();
  }
}

void llvm_profile_reset_counters_wrapper(void) {
  if (__llvm_profile_reset_counters) {
    __llvm_profile_reset_counters();
  }
}
