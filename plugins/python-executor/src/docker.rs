use bollard::auth::DockerCredentials;
use bollard::image::CreateImageOptions;
use bollard::models::CreateImageInfo;
use bollard::Docker;

use thiserror::Error;
use tokio_stream::StreamExt;

#[derive(Error, Debug)]
enum DockerError {
    #[error("Failed to connect to docker: {0}")]
    ConnectionFailure(bollard::errors::Error),

    #[error("Failed to pull image: {0}")]
    PullError(bollard::errors::Error),
}

struct LocalDocker {
    docker: Docker,
    username: Option<String>,
    password: Option<String>,
}

impl LocalDocker {
    fn new() -> Result<Self, DockerError> {
        let d =
            Docker::connect_with_local_defaults().map_err(|e| DockerError::ConnectionFailure(e))?;

        Ok(Self {
            docker: d,
            username: Some(String::from("")),
            password: Some(String::from("")),
        })
    }

    // pull, given a image name, will attempt to pull an image from the docker registry
    async fn pull(&mut self, image: &str) -> Result<(), DockerError> {
        let stream = self.docker.create_image(
            Some(CreateImageOptions {
                from_image: image,
                ..Default::default()
            }),
            None,
            make_credentials(&self.username, &self.password),
        );

        stream
            .collect::<Vec<Result<CreateImageInfo, bollard::errors::Error>>>()
            .await
            .pop()
            .unwrap()
            .map_err(|e| DockerError::PullError(e))
            .map(|_| ())
    }
}

fn make_credentials(
    username: &Option<String>,
    password: &Option<String>,
) -> Option<DockerCredentials> {
    Some(DockerCredentials {
        username: username.to_owned(),
        password: password.to_owned(),
        ..Default::default()
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn connects_ok() {
        assert!(!LocalDocker::new().is_err());
    }

    #[tokio::test]
    async fn pulls_fail() {
        let mut d = LocalDocker::new().unwrap();
        let res = d.pull("ubuntu:madeup").await;
        assert!(res.is_err());
    }

    #[tokio::test]
    async fn pulls_ok() {
        let mut d = LocalDocker::new().unwrap();
        let res = d.pull("ubuntu:kinetic").await;
        assert!(res.is_ok());
    }
}
