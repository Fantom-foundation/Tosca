unsafe extern "C" {
    safe fn llvm_profile_enabled_wrapper() -> u8;
    safe fn llvm_profile_set_filename_wrapper(filename: *const std::ffi::c_char);
    safe fn llvm_profile_write_file_wrapper();
    safe fn llvm_profile_reset_counters_wrapper();
}

pub fn llvm_profile_enabled() -> u8 {
    llvm_profile_enabled_wrapper()
}

pub fn llvm_profile_set_filename(filename: Option<&std::ffi::c_char>) {
    llvm_profile_set_filename_wrapper(filename.map(|f| &raw const *f).unwrap_or(std::ptr::null()));
}

pub fn llvm_profile_write_file() {
    llvm_profile_write_file_wrapper();
}

pub fn llvm_profile_reset_counters() {
    llvm_profile_reset_counters_wrapper();
}
