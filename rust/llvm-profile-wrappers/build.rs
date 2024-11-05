use std::{env, process::Command};

fn main() {
    let source = "src/llvm_profile_wrappers.c";
    let out_dir = env::var("OUT_DIR").unwrap();

    let object_file = format!("{}/llvm_profile_wrappers.o", out_dir);
    let output = Command::new("gcc")
        .args(["-c", source, "-o", &object_file])
        .output()
        .expect("Failed to compile C code");

    if !output.status.success() {
        panic!(
            "C compilation failed: {}",
            String::from_utf8_lossy(&output.stderr)
        );
    }

    let static_lib = format!("{}/libllvm_profile_wrappers.a", out_dir);
    let output = Command::new("ar")
        .args(["crus", &static_lib, &object_file])
        .output()
        .expect("Failed to create static library");

    if !output.status.success() {
        panic!(
            "Static library creation failed: {}",
            String::from_utf8_lossy(&output.stderr)
        );
    }

    println!("cargo::rerun-if-changed={}", source);
    println!("cargo::rustc-link-lib=static=llvm_profile_wrappers");
    println!("cargo::rustc-link-search=native={}", out_dir);
}
