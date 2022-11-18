use std::env;
use std::process;
use std::io;
use std::io::Read;
use std::io::Write;
use std::fs::File;
use std::error::Error;

fn cat(path: &String) -> Result<(), Box<dyn Error>> {
    let mut input = File::open(path)?;
    let mut buffer = [0; 256];
    let mut output = io::stdout();
    
    loop {
        let read = input.read(&mut buffer)?;
        let mut written = 0;

        if read <= 0 { 
            return Ok(());
        }

        loop {
            written = output.write(&mut buffer[written..read])?;
            if written >= read { break; }
        }
    }
}

fn main() {
    let args: Vec<String> = env::args().collect();
    for arg in args.iter().skip(1) {
        if let Err(err) = cat(arg) {
            eprintln!("{}", err);
            process::exit(1);
        }
    }
    
    process::exit(0);
}
