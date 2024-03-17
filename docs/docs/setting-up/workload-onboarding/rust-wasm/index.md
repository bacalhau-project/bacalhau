---
sidebar_label: "Rust WASM"
sidebar_position: 10
---
# Running Rust programs as WebAssembly (WASM)


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

Bacalhau supports running jobs as a [WebAssembly (WASM)](https://webassembly.org/) program. This example demonstrates how to compile a [Rust](https://www.rust-lang.org/) project into WebAssembly and run the program on Bacalhau.

### Prerequisites

1. To get started, you need to install the Bacalhau client, see more information [here](../../../getting-started/installation.md).

2. A working Rust installation with the `wasm32-wasi` target. For example, you can use [`rustup`](https://rustup.rs/) to install Rust and configure it to build WASM targets. For those using the notebook, these are installed in hidden cells below.

## 1. Develop a Rust Program Locally

We can use `cargo` (which will have been installed by `rustup`) to start a new project (`my-program`) and compile it:


```bash
%%bash
cargo init my-program
```


We can then write a Rust program. Rust programs that run on Bacalhau can read and write files, access a simple clock, and make use of pseudo-random numbers. They cannot memory-map files or run code on multiple threads.

The program below will use the Rust `imageproc` crate to resize an image through seam carving, based on [an example from their repository](https://github.com/image-rs/imageproc/blob/master/examples/seam_carving.rs).


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

In the main function `main()` an image is loaded, the original is saved, and then a loop is performed to reduce the width of the image by removing "seams." The results of the process are saved, including the original image with drawn seams and a gradient image with highlighted seams.

We also need to install the `imageproc` and `image` libraries and switch off the default features to make sure that multi-threading is disabled (`default-features = false`). After disabling the default features, you need to explicitly specify only the features that you need:


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

We can now build the Rust program into a WASM blob using `cargo`:


```bash
%%bash
cd my-program && cargo build --target wasm32-wasi --release
```
This command navigates to the `my-program` directory and builds the project using Cargo with the target set to `wasm32-wasi` in release mode.

This will generate a WASM file at `./my-program/target/wasm32-wasi/release/my-program.wasm` which can now be run on Bacalhau.

## 2. Running WASM on Bacalhau
Now that we have a WASM binary, we can upload it to IPFS and use it as input to a Bacalhau job.

The `-i` flag allows specifying a URI to be mounted as a named volume in the job, which can be an IPFS CID, HTTP URL, or S3 object.

For this example, we are using an image of the Statue of Liberty that has been pinned to a storage facility.


```bash
%%bash --out job_id
bacalhau wasm run ./my-program/target/wasm32-wasi/release/my-program.wasm _start \
    --id-only \
    -i ipfs://bafybeifdpl6dw7atz6uealwjdklolvxrocavceorhb3eoq6y53cbtitbeu:/inputs
```


### Structure of the Commands

`bacalhau wasm run`: call to Bacalhau

`./my-program/target/wasm32-wasi/release/my-program.wasm`: the path to the WASM file that will be executed

` _start`: the entry point of the WASM program, where its execution begins

`--id-only`: this flag indicates that only the identifier of the executed job should be returned

`-i ipfs://bafybeifdpl6dw7atz6uealwjdklolvxrocavceorhb3eoq6y53cbtitbeu:/inputs`: input data volume that will be accessible within the job at the specified destination path

When a job is submitted, Bacalhau prints out the related job_id. We store that in an environment variable so that we can reuse it later on:

```python
%env JOB_ID={job_id}
```





You can download your job results directly by using `bacalhau get`. Alternatively, you can choose to create a directory to store your results. In the command below, we created a directory (`wasm_results`) and downloaded our job output to be stored in that directory.

We can now get the results.



```bash
%%bash
rm -rf wasm_results && mkdir -p wasm_results
bacalhau get ${JOB_ID} --output-dir wasm_results
```

## Viewing Job Output

When we view the files, we can see the original image, the resulting shrunk image, and the seams that were removed.

```python
import IPython.display as display
display.Image("./wasm_results/outputs/original.png")
```





![png](index_files/index_18_0.png)





```python
display.Image("./wasm_results/outputs/annotated_gradients.png")
```





![png](index_files/index_19_0.png)





```python
display.Image("./wasm_results/outputs/shrunk.png")
```





![png](index_files/index_20_0.png)


## Support
If you have questions or need support or guidance, please reach out to the [Bacalhau team via Slack](https://bacalhauproject.slack.com/ssb/redirect) (**#general** channel).
