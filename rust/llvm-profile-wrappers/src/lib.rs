extern "C" {
    fn llvm_profile_enabled_wrapper() -> u8;
    fn llvm_profile_set_filename_wrapper(filename: *const std::ffi::c_char);
    fn llvm_profile_write_file_wrapper();
    fn llvm_profile_reset_counters_wrapper();
}

pub fn llvm_profile_enabled() -> u8 {
    unsafe { llvm_profile_enabled_wrapper() }
}

/// # Safety
/// The provided filename can be a C string to set a new name or null to reset to the default
/// behavior.
pub unsafe fn llvm_profile_set_filename(filename: *const std::ffi::c_char) {
    llvm_profile_set_filename_wrapper(filename);
}

pub fn llvm_profile_write_file() {
    unsafe {
        llvm_profile_write_file_wrapper();
    }
}

pub fn llvm_profile_reset_counters() {
    unsafe {
        llvm_profile_reset_counters_wrapper();
    }
}
