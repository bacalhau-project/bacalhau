use std::process;
use std::env;

fn main() {
    let code = env::var("EXIT_CODE").unwrap_or("0".to_owned()).
        parse::<i32>().ok().expect("Must be an integer");
    println!("Exiting with {}", code);
    process::exit(code)
}
