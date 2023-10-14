fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = "./src";

    tonic_build::configure()
        .protoc_arg("--experimental_allow_proto3_optional") // for older systems
        .build_client(false)
        .build_server(true)
        .out_dir(out_dir)
        .file_descriptor_set_path(format!("{out_dir}/executor.bin"))
        .compile(&["proto/executor.proto"], &["proto"])?;

    Ok(())
}
