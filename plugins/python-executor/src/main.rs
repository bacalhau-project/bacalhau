pub mod btypes;
pub mod executor;
pub mod service;

use std::env;

use executor::executor_server::ExecutorServer;
use service::ExecutorService;

use tokio::io::{self, AsyncWriteExt};
use tonic::transport::Server;

pub(crate) const FILE_DESCRIPTOR_SET: &[u8] = include_bytes!("executor.bin");

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let port = get_port();
    let address = format!("127.0.0.1:{port}").parse().unwrap();
    let executor_service = ExecutorService::default();

    write_handshake(port).await;

    let reflection_service = tonic_reflection::server::Builder::configure()
        .register_encoded_file_descriptor_set(FILE_DESCRIPTOR_SET)
        .build()
        .unwrap();

    Server::builder()
        .add_service(reflection_service)
        .add_service(ExecutorServer::new(executor_service))
        .serve(address)
        .await?;

    Ok(())
}

fn get_port() -> i32 {
    env::var("PYTHON_EXECUTOR_PORT").map_or(2112, |v| v.parse::<i32>().unwrap())
}

async fn write_handshake(port: i32) {
    print!("1|1|tcp|127.0.0.1:{port}|grpc\n");
    io::stdout().flush().await.unwrap();
}
