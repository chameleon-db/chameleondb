use std::env;
use std::path::PathBuf;

fn main() {
    println!("cargo:warning=Build script starting...");
    
    // Generate LALRPOP parser
    let root_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap());
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());
    let parser_dir = root_dir.join("src/parser");
    
    println!("cargo:warning=Processing LALRPOP files in: {:?}", parser_dir);
    println!("cargo:warning=Output dir: {:?}", out_dir);
    
    lalrpop::Configuration::new()
        .set_out_dir(out_dir)
        .process_dir(parser_dir)
        .expect("Failed to process LALRPOP files");
    
    println!("cargo:warning=LALRPOP processing complete!");
    
    // Generate C header (cbindgen)
    let crate_dir = env::var("CARGO_MANIFEST_DIR").unwrap();
    
    cbindgen::Builder::new()
        .with_crate(crate_dir)
        .with_language(cbindgen::Language::C)
        .with_include_guard("CHAMELEON_H")
        .generate()
        .expect("Unable to generate bindings")
        .write_to_file("include/chameleon.h");
}