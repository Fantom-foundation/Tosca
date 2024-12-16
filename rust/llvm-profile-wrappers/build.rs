use std::env;

fn main() {
    let source = "src/llvm_profile_wrappers.c";
    let out_dir = env::var("OUT_DIR").unwrap();

    let lib_name = "llvm_profile_wrappers";
    cc::Build::new().file(source).compile(lib_name);

    println!("cargo::rerun-if-changed={source}");
    println!("cargo::rustc-link-lib=static={lib_name}");
    println!("cargo::rustc-link-search=native={out_dir}");
}
