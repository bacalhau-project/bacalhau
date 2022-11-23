---
sidebar_label: "Rust - WASM"
sidebar_position: 15
---
# Running Rust programs as WebAssembly (WASM)

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/rust-wasm/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/rust-wasm/index.ipynb)

Bacalhau supports running jobs as a [WebAssembly (WASM)](https://webassembly.org/) program rather than using a Docker container. This examples demonstrates how to compile a [Rust](https://www.rust-lang.org/) project into WebAssembly and run the program on Bacalhau.

### Prerequisites

* Make sure you have the latest `bacalhau` client installed by following the [getting started instructions](../../../getting-started/installation).
* You can use [`rustup`](https://rustup.rs/) to install Rust and configure it to build WASM targets.

## 1. Develop a Rust Program Locally

We can use `cargo` (which will have been installed by `rustup`) to start a new project and compile it. 


```bash
cargo init my-program
```

         Created binary (application) package


We can then write a Rust program. Rust programs that run on Bacalhau can read and write files, access a simple clock and make use of psudeo-random numbers. They cannot memory-map files or run code on multiple threads.

The below program will make use of the Rust `imageproc` crate to resize an image through seam carving, based on [an example from their repository](https://github.com/image-rs/imageproc/blob/master/examples/seam_carving.rs).


```python
%%writefile ./my-program/src/main.rs 
use image::{open, GrayImage, Luma, Pixel};
use imageproc::definitions::Clamp;
use imageproc::gradients::sobel_gradient_map;
use imageproc::map::map_colors;
use imageproc::seam_carving::*;
use std::path::Path;

fn main() {
    let input_path = "inputs/image0.JPG";
    let output_dir = "outputs/";

    let input_path = Path::new(&input_path);
    let output_dir = Path::new(&output_dir);

    // Load image and convert to grayscale
    let input_image = open(input_path)
        .expect(&format!("Could not load image at {:?}", input_path))
        .to_rgb8();

    // Save original image in output directory
    let original_path = output_dir.join("original.png");
    input_image.save(&original_path).unwrap();

    // We will reduce the image width by this amount, removing one seam at a time.
    let seams_to_remove: u32 = input_image.width() / 6;

    let mut shrunk = input_image.clone();
    let mut seams = Vec::new();

    // Record each removed seam so that we can draw them on the original image later.
    for i in 0..seams_to_remove {
        if i % 100 == 0 {
            println!("Removing seam {}", i);
        }        
        let vertical_seam = find_vertical_seam(&shrunk);
        shrunk = remove_vertical_seam(&mut shrunk, &vertical_seam);
        seams.push(vertical_seam);
    }

    // Draw the seams on the original image.
    let gray_image = map_colors(&input_image, |p| p.to_luma());
    let annotated = draw_vertical_seams(&gray_image, &seams);
    let annotated_path = output_dir.join("annotated.png");
    annotated.save(&annotated_path).unwrap();

    // Draw the seams on the gradient magnitude image.
    let gradients = sobel_gradient_map(&input_image, |p| {
        let mean = (p[0] + p[1] + p[2]) / 3;
        Luma([mean as u32])
    });
    let clamped_gradients: GrayImage = map_colors(&gradients, |p| Luma([Clamp::clamp(p[0])]));
    let annotated_gradients = draw_vertical_seams(&clamped_gradients, &seams);
    let gradients_path = output_dir.join("gradients.png");
    clamped_gradients.save(&gradients_path).unwrap();
    let annotated_gradients_path = output_dir.join("annotated_gradients.png");
    annotated_gradients.save(&annotated_gradients_path).unwrap();

    // Save the shrunk image.
    let shrunk_path = output_dir.join("shrunk.png");
    shrunk.save(&shrunk_path).unwrap();
}
```

    Overwriting ./my-program/src/main.rs


We also need to install the `imageproc` and `image` libraries and switch off the default features to make sure that multi-threading is disabled.


```python
%%writefile ./my-program/Cargo.toml
[package]
name = "my-program"
version = "0.1.0"
edition = "2021"

[dependencies.image]
version = "0.24.4"
default-features = false
features = ["png", "jpeg", "bmp"]

[dependencies.imageproc]
version = "0.23.0"
default-features = false
```

    Overwriting ./my-program/Cargo.toml


We can now build the Rust program into a WASM blob using `cargo`.


```bash
rustup target add wasm32-wasi
cd my-program && cargo build --target wasm32-wasi --release
```

This will generate a WASM file at `./my-program/target/wasm32-wasi/my-program.wasm` which can now be run on Bacalhau.

## 2. Running WASM on Bacalhau
Now that we have a WASM binary, we can upload it to IPFS and use it as input to a Bacalhau job.

The -v switch allows specifying an IPFS CID to mount as a named volume in the job. There is also a -u switch which can download inputs via HTTP.

For this example, we are using an image of the Statue of Liberty that has been pinned to a storage facility.


```bash
bacalhau wasm run ./my-program/target/wasm32-wasi/release/my-program.wasm _start \
    -v bafybeifdpl6dw7atz6uealwjdklolvxrocavceorhb3eoq6y53cbtitbeu:inputs | tee job.txt
```

    Uploading "./my-program/target/wasm32-wasi/release/my-program.wasm" to server to execute command in context, press Ctrl+C to cancel
    Job successfully submitted. Job ID: 1cf373a0-bb75-4b68-8c7e-c2e0e10d7eaa
    Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):
    
    	       Creating job for submission ... done ✅
    	       Finding node(s) for the job ... done ✅
    	             Node accepted the job ... done ✅
    	   Job finished, verifying results ... done ✅
    	      Results accepted, publishing ... Results CID: QmaxyTrc3zSb6ggUVYgXb9yxVJZ9cXv6Y6u55Czm2eqaWD
    Job Results By Node:
    Node QmSyJ8VU:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    Node QmVAb7r2:
      Shard 0:
        Status: Completed
        Container Exit Code: -1
        Stdout:
          Removing seam 0
    Removing seam 100
    
        Stderr: <NONE>
    Node QmXaXu9N:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    Node QmYgxZiy:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    Node QmdZQ7Zb:
      Shard 0:
        Status: Cancelled
        No RunOutput for this shard
    
    To download the results, execute:
      bacalhau get 1cf373a0-bb75-4b68-8c7e-c2e0e10d7eaa
    
    To get more details about the run, execute:
      bacalhau describe 1cf373a0-bb75-4b68-8c7e-c2e0e10d7eaa


We can now get the results. When we view the files, we can see the original image, the resulting shrunk image, and the seams that were removed.


```bash
mkdir -p wasm_results
bacalhau get $(grep "Job ID:" job.txt | cut -f2 -d:) --output-dir wasm_results
```

    Fetching results of job '1cf373a0-bb75-4b68-8c7e-c2e0e10d7eaa'...
    [90m18:29:54.069 |[0m [1m[31mERR[0m[0m [1mipfs/node.go:178[0m[36m >[0m ipfs node failed to serve API: failed to listen on api multiaddr: listen tcp4 127.0.0.1:5001: bind: address already in use
    Results for job '1cf373a0-bb75-4b68-8c7e-c2e0e10d7eaa' have been written to...
    wasm_results



```python
import IPython.display as display
display.Image("./wasm_results/combined_results/outputs/original.png")
```




    
![png](index_files/index_16_0.png)
    




```python
display.Image("./wasm_results/combined_results/outputs/annotated_gradients.png")
```




    
![png](index_files/index_17_0.png)
    




```python
display.Image("./wasm_results/combined_results/outputs/shrunk.png")
```




    
![png](index_files/index_18_0.png)
    


