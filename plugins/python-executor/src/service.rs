use std::pin::Pin;

use crate::btypes;
use crate::executor::executor_server::Executor;
use crate::executor::{
    CancelCommandRequest, CancelCommandResponse, IsInstalledRequest, IsInstalledResponse,
    OutputStreamRequest, OutputStreamResponse, RunCommandRequest, RunCommandResponse,
    ShouldBidBasedOnUsageRequest, ShouldBidRequest, ShouldBidResponse, StartResponse, WaitRequest,
};

use tokio_stream::Stream;
use tonic::{Request, Response, Status};

#[derive(Debug, Default)]
pub struct ExecutorService {}

#[tonic::async_trait]
impl Executor for ExecutorService {
    type WaitStream = Pin<Box<dyn Stream<Item = Result<RunCommandResponse, Status>> + Send>>;
    type GetOutputStreamStream =
        Pin<Box<dyn Stream<Item = Result<OutputStreamResponse, Status>> + Send>>;

    async fn start(
        &self,
        _request: Request<RunCommandRequest>,
    ) -> Result<Response<StartResponse>, tonic::Status> {
        todo!()
    }

    async fn run(
        &self,
        _request: Request<RunCommandRequest>,
    ) -> Result<Response<RunCommandResponse>, tonic::Status> {
        // Convert req.params into btypes::RunCommandRequest, should be
        // let rcr = _request.get_ref();
        let req = _request.get_ref();
        let bytes = String::from_utf8(req.params.clone()).unwrap();
        let rcr = serde_json::from_str::<btypes::RunCommandRequest>(&bytes).unwrap();

        println!("Execution ID: {}", rcr.execution_id);

        // Convert our response back, should be
        // let resp = btypes::RunCommandResponse::new().with_exit_code(0);
        // Ok(Response::new(resp))
        let resp = btypes::RunCommandResponse::new().with_exit_code(0);
        let v = serde_json::to_vec(&resp);

        let response: RunCommandResponse = RunCommandResponse { params: v.unwrap() };
        Ok(Response::new(response))
    }

    async fn wait(
        &self,
        _request: Request<WaitRequest>,
    ) -> Result<Response<Self::WaitStream>, Status> {
        todo!()
    }

    async fn cancel(
        &self,
        _request: Request<CancelCommandRequest>,
    ) -> Result<Response<CancelCommandResponse>, Status> {
        todo!()
    }

    async fn is_installed(
        &self,
        _request: tonic::Request<IsInstalledRequest>,
    ) -> Result<Response<IsInstalledResponse>, Status> {
        let resp = IsInstalledResponse { installed: true };
        Ok(Response::new(resp))
    }

    async fn should_bid(
        &self,
        _request: Request<ShouldBidRequest>,
    ) -> Result<Response<ShouldBidResponse>, Status> {
        todo!()
    }

    async fn should_bid_based_on_usage(
        &self,
        _request: Request<ShouldBidBasedOnUsageRequest>,
    ) -> Result<Response<ShouldBidResponse>, Status> {
        todo!()
    }

    async fn get_output_stream(
        &self,
        _request: Request<OutputStreamRequest>,
    ) -> Result<Response<Self::GetOutputStreamStream>, Status> {
        todo!()
    }
}
